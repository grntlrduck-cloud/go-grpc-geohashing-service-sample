package dynamo

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gocarina/gocsv"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/domain/poi"
)

const (
	routeHashesLimit     = 200
	bboxHashesLimit      = 150
	proxHashesLimit      = 150
	dynamoMaxBatchSize   = 10
	maxConcurrentQueries = 10 // Configurable max concurrent queries
	testInitDataPath     = "config/db/local/cpoi_dynamo_items_int_test.csv"
)

type PoIGeoRepository struct {
	dynamoClient    DBClient
	tableName       string
	createInitTable bool
	initDataPath    string
}

type PoIGeoRepositoryOptions func(p *PoIGeoRepository)

func WithDynamoClientWrapper(client DBClient) PoIGeoRepositoryOptions {
	return func(p *PoIGeoRepository) {
		p.dynamoClient = client
	}
}

func WithTableName(tableName string) PoIGeoRepositoryOptions {
	return func(p *PoIGeoRepository) {
		p.tableName = tableName
	}
}

func WithCreateAndInitTable(createAndInitTable bool) PoIGeoRepositoryOptions {
	return func(p *PoIGeoRepository) {
		p.createInitTable = createAndInitTable
	}
}

func WithTestInitDataOverrid(testInitDataPath string) PoIGeoRepositoryOptions {
	return func(p *PoIGeoRepository) {
		p.initDataPath = testInitDataPath
	}
}

func NewPoIGeoRepository(
	logger *zap.Logger,
	opts ...PoIGeoRepositoryOptions,
) (poi.Repository, error) {
	repo := &PoIGeoRepository{
		tableName:    "NOT_DEFINED",
		initDataPath: testInitDataPath,
	}
	for _, opt := range opts {
		opt(repo)
	}
	if repo.dynamoClient == nil {
		cl, err := NewClientWrapper()
		if err != nil {
			return nil, fmt.Errorf("dyanmo client was nil but failed to initialize: %w", err)
		}
		repo.dynamoClient = cl
	}
	if repo.createInitTable {
		err := repo.createTableAndLoadData(logger)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to set up table with test data from csv for local testing: %w",
				err,
			)
		}
	}
	return repo, nil
}

func (pgr *PoIGeoRepository) UpsertBatch(
	ctx context.Context,
	pois []*poi.PoILocation,
	logger *zap.Logger,
) error {
	// handle context cancelation
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// verify validity
	if len(pois) == 0 {
		logger.Warn(
			"skipping batch upsert because pois is empty slice",
		)
		return nil
	}

	// map domain model to dynamo items and marshall to AttributeValues
	// and assemble list of WriteRequests
	chunks, err := createBatchRequests(pois)
	if err != nil {
		logger.Error(
			"failed to create batch requests",
			zap.Error(err),
		)
		return poi.ErrDBEntityMapping
	}

	// upsert chunks
	var errs []error
	for i, c := range chunks {
		if len(c) == 0 {
			logger.Warn("skipping to upsert chunk since it is empty",
				zap.Int("batch_num", i),
				zap.Int("total_num_batches", len(chunks)),
				zap.Int("num_items", len(c)),
			)
			continue
		}
		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{pgr.tableName: c},
		}
		_, err := pgr.dynamoClient.BatchPutItem(ctx, input)
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			_, err := pgr.dynamoClient.BatchPutItem(ctx, input)
			if err != nil {
				logger.Error(
					"failed to perform batch PutItem after retry",
					zap.Int("batch_num", i),
					zap.Int("total_num_batches", len(chunks)),
					zap.Int("num_items", len(c)),
					zap.Error(err),
				)
				errs = append(errs, err)
			}
		}
		logger.Debug(
			"successfully inserted batch",
			zap.Int("batch_num", i),
			zap.Int("total_num_batches", len(chunks)),
			zap.Int("num_items", len(c)),
		)
		time.Sleep(50 * time.Microsecond)
	}
	if len(errs) > 0 {
		logger.Error("batch upsert incomplete",
			zap.Error(errs[0]),
			zap.Int("num_failed_batches", len(errs)),
		)
		return poi.ErrDBBatchUpsert
	}
	return nil
}

func (pgr *PoIGeoRepository) Upsert(
	ctx context.Context,
	domain *poi.PoILocation,
	logger *zap.Logger,
) error {
	// handle context cancelation
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// map domain to db item
	item, err := NewItemFromDomain(domain)
	if err != nil {
		logger.Warn("invalid coordinate for proximity search",
			zap.Error(err),
		)
		return fmt.Errorf("unable to upsert location: %w", err)
	}
	// map db item to dynamodb attributevalues
	avs, err := attributevalue.MarshalMap(&item)
	if err != nil {
		logger.Error("unable to marshall item to DynamoDB AttributeValues",
			zap.Error(err),
		)
		return poi.ErrDBEntityMapping
	}
	// create PutItemInput and perform request
	putItemInput := &dynamodb.PutItemInput{
		Item:      avs,
		TableName: &pgr.tableName,
	}
	_, err = pgr.dynamoClient.PutItem(ctx, putItemInput)
	if err != nil {
		logger.Error("failed to PutItem",
			zap.Error(err),
		)
		return poi.ErrDBUpsert
	}
	return nil
}

func (pgr *PoIGeoRepository) GetByID(
	ctx context.Context,
	id ksuid.KSUID,
	logger *zap.Logger,
) (*poi.PoILocation, error) {
	// handle context cancelation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// create GetItemInput
	getItemInput := &dynamodb.GetItemInput{
		TableName: aws.String(pgr.tableName),
		Key: map[string]types.AttributeValue{
			CPoIItemPK: &types.AttributeValueMemberS{Value: id.String()},
		},
	}
	// check query output and marshall to domain
	output, err := pgr.dynamoClient.GetItem(ctx, getItemInput)
	if err != nil {
		logger.Error("failed to GetItem",
			zap.Error(err),
		)
		return nil, err
	}
	item := new(CPoIItem)
	err = attributevalue.UnmarshalMap(output.Item, item)
	if err != nil {
		logger.Error("failed to unmarshal GetItem output",
			zap.Error(err),
		)
		return nil, err
	}
	// handle location not found since dynamo does not error
	if item.Pk == "" {
		return nil, poi.ErrLocationNotFound
	}
	return item.Domain()
}

func (pgr *PoIGeoRepository) GetByProximity(
	ctx context.Context,
	cntr poi.Coordinates,
	radius float64,
	logger *zap.Logger,
) ([]*poi.PoILocation, error) {
	// handle context cancelation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// create hashes, validate, and check if we can perform the search without major performance cuts
	hashes, err := newHashesFromRadiusCenter(cntr, radius, nil)
	if err != nil {
		return nil, poi.ErrInvalidSearchCoordinates
	}
	// google s2 does not guarantee that the set MaxCells can be fulfilled
	// an arbitrary large list of hashes might be returned
	if len(hashes) > proxHashesLimit {
		logger.Error("too many hashes calculated for proximity",
			zap.Int("num_hashes", len(hashes)),
		)
		return nil, poi.ErrTooLargeSearchArea
	}
	// perform parallel queries
	res, err := pgr.parallelQueryHashes(ctx, logger, hashes)
	if err != nil {
		logger.Error("failed to query by proximity",
			zap.Error(err),
		)
		return nil, poi.ErrDBQuery
	}
	return res, nil
}

func (pgr *PoIGeoRepository) GetByBbox(
	ctx context.Context,
	sw, ne poi.Coordinates,
	logger *zap.Logger,
) ([]*poi.PoILocation, error) {
	// handle context cancelation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// create hashes, validate, and check if we can perform the search without major performance cuts
	hashes, err := newHashesFromBbox(ne, sw, nil)
	if err != nil {
		logger.Warn("invalid coordinates for bounding box",
			zap.Error(err),
		)
		return nil, poi.ErrInvalidSearchCoordinates
	}
	// google s2 does not guarantee that the set MaxCells can be fulfilled
	// an arbitrary large list of hashes might be returned
	if len(hashes) > bboxHashesLimit {
		logger.Error("too many hashes calculated for bbox",
			zap.Int("num_hashes", len(hashes)),
		)
		return nil, poi.ErrTooLargeSearchArea
	}
	res, err := pgr.parallelQueryHashes(ctx, logger, hashes)
	if err != nil {
		logger.Error("failed to query by bbox",
			zap.Error(err),
		)
		return nil, poi.ErrDBQuery
	}
	return res, nil
}

func (pgr *PoIGeoRepository) GetByRoute(
	ctx context.Context,
	path []poi.Coordinates,
	logger *zap.Logger,
) ([]*poi.PoILocation, error) {
	// handle context cancelation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// create hashes, validate, and check if we can perform the search without major performance cuts
	hashes, err := newHashesFromRoute(path, nil)
	if err != nil {
		logger.Warn("invalid coordinates in provided coordinate path",
			zap.Error(err),
		)
		return nil, poi.ErrInvalidSearchCoordinates
	}
	// google s2 does not guarantee that the set MaxCells can be fulfilled
	// an arbitrary large list of hashes might be returned
	if len(hashes) > routeHashesLimit {
		logger.Error("too many hashes calculated for route",
			zap.Int("num_hashes", len(hashes)),
		)
		return nil, poi.ErrTooLargeSearchArea
	}
	res, err := pgr.parallelQueryHashes(ctx, logger, hashes)
	if err != nil {
		logger.Error("failed to query by route",
			zap.Error(err),
		)
		return nil, poi.ErrDBQuery
	}
	return res, nil
}

func (pgr *PoIGeoRepository) parallelQueryHashes(
	ctx context.Context,
	logger *zap.Logger,
	hashes []geoHash,
) ([]*poi.PoILocation, error) {
	queries := pgr.queryInputFromHashes(hashes)
	logger.Info("sending parallel requests for geo query",
		zap.Int("queries", len(queries)),
	)

	// Use buffered channel with exact size needed
	resC := make(chan poiQueryResult, len(queries))
	errGrp, gctx := errgroup.WithContext(ctx)

	// Set reasonable concurrency limit
	workers := min(len(queries), maxConcurrentQueries)
	errGrp.SetLimit(workers)

	// Launch workers
	for i := range queries {
		query := queries[i] // Create new variable to avoid closure issues
		errGrp.Go(func() error {
			qres := pgr.query(gctx, query)
			if qres.err != nil {
				logger.Error("query failed",
					zap.Error(qres.err),
				)
				return qres.err
			}
			if len(qres.pois) == 0 {
				return nil
			}

			// Try to send results with timeout
			select {
			case resC <- qres:
				return nil
			case <-gctx.Done():
				return gctx.Err()
			case <-time.After(3 * time.Second):
				return fmt.Errorf(
					"timeout writing result during parallel db queries",
				)
			}
		})
	}

	// Close results channel when all workers complete
	go func() {
		_ = errGrp.Wait()
		close(resC)
	}()

	// Collect results with pre-allocated slice
	pois := make([]*poi.PoILocation, 0, len(queries)*2) // Estimate capacity
	for r := range resC {
		logger.Debug(
			"appending pois from parallel query results",
			zap.Int("num_results", len(r.pois)),
		)
		pois = append(pois, r.pois...)
	}

	// Check for worker errors
	if err := errGrp.Wait(); err != nil {
		return nil, fmt.Errorf("parallel query failed: %w", err)
	}

	return pois, nil
}

type poiQueryResult struct {
	pois []*poi.PoILocation
	err  error
}

func (pgr *PoIGeoRepository) query(
	ctx context.Context,
	input *dynamodb.QueryInput,
) poiQueryResult {
	items := make([]map[string]types.AttributeValue, 0)
	queryResult, err := pgr.dynamoClient.QueryItem(ctx, input)
	if err != nil {
		return poiQueryResult{nil, fmt.Errorf("failed to query PoIs: %w", err)}
	}
	items = append(items, queryResult.Items...)
	for queryResult.LastEvaluatedKey != nil {
		input.ExclusiveStartKey = queryResult.LastEvaluatedKey
		res, errQ := pgr.dynamoClient.QueryItem(ctx, input)
		if errQ != nil {
			return poiQueryResult{nil, fmt.Errorf("failed to call query page: %w", errQ)}
		}
		items = append(items, res.Items...)
	}
	domain, err := mapAvs(items)
	if err != nil {
		return poiQueryResult{nil, fmt.Errorf("failed to map results: %w", err)}
	}
	return poiQueryResult{domain, nil}
}

func (pgr *PoIGeoRepository) queryInputFromHashes(hashes []geoHash) []*dynamodb.QueryInput {
	queries := make([]*dynamodb.QueryInput, 0)
	for _, v := range hashes {
		keyCondition := fmt.Sprintf(
			"%s = :pk AND %s BETWEEN :skmin AND :skmax",
			CPoIItemGeoIndexPK,
			CPoIItemGeoIndexSK,
		)
		query := &dynamodb.QueryInput{
			TableName:              aws.String(pgr.tableName),
			IndexName:              aws.String(CPoIItemGeoIndexName),
			KeyConditionExpression: aws.String(keyCondition),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberN{
					Value: strconv.FormatUint(v.trimmed(CPoIItemCellLevel), 10),
				},
				":skmin": &types.AttributeValueMemberN{Value: strconv.FormatUint(v.min(), 10)},
				":skmax": &types.AttributeValueMemberN{Value: strconv.FormatUint(v.max(), 10)},
			},
		}
		queries = append(queries, query)
	}
	return queries
}

func (pgr *PoIGeoRepository) createTableAndLoadData(logger *zap.Logger) error {
	logger.Warn("table will be created and initialized with initial data!")
	err := pgr.createInitPoiTable()
	if err != nil {
		return fmt.Errorf("failed to initialize table for local testing: %w", err)
	}
	logger.Info("created table successfully")

	time.Sleep(100 * time.Millisecond)

	logger.Info("upserting test data")
	err = pgr.loadInitData(logger)
	if err != nil {
		return fmt.Errorf("failed to load initial data into table for testing: %w", err)
	}
	logger.Info("successfully created table and loaded test data")
	return nil
}

func (pgr *PoIGeoRepository) createInitPoiTable() error {
	input := dynamodb.CreateTableInput{
		TableName: aws.String(pgr.tableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(CPoIItemPK), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String(CPoIItemPK), AttributeType: types.ScalarAttributeTypeS},
			{
				AttributeName: aws.String(CPoIItemGeoIndexPK),
				AttributeType: types.ScalarAttributeTypeN,
			},
			{
				AttributeName: aws.String(CPoIItemGeoIndexSK),
				AttributeType: types.ScalarAttributeTypeN,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String(CPoIItemGeoIndexName),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String(CPoIItemGeoIndexPK), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String(CPoIItemGeoIndexSK), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(10),
					WriteCapacityUnits: aws.Int64(10),
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}
	_, err := pgr.dynamoClient.CreateTable(context.Background(), &input)
	if err != nil {
		return fmt.Errorf("failed to perform create table request: %w", err)
	}
	return nil
}

func createBatchRequests(pois []*poi.PoILocation) ([][]types.WriteRequest, error) {
	rqsts := make([]types.WriteRequest, len(pois))
	for i, v := range pois {
		item, err := NewItemFromDomain(v)
		if err != nil {
			return nil, fmt.Errorf("unable to map item to domain: %w", err)
		}
		av, err := attributevalue.MarshalMap(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to markshall item: %w", err)
		}
		rqsts[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: av}}
	}
	numChunks := (len(pois) + dynamoMaxBatchSize - 1) / dynamoMaxBatchSize
	chunks := make([][]types.WriteRequest, 0, numChunks)

	// create chunks with max 25 items
	for dynamoMaxBatchSize < len(rqsts) {
		rqsts, chunks = rqsts[dynamoMaxBatchSize:], append(
			chunks,
			rqsts[0:dynamoMaxBatchSize:dynamoMaxBatchSize],
		)
	}
	chunks = append(chunks, rqsts)
	return chunks, nil
}

func (pgr *PoIGeoRepository) loadInitData(logger *zap.Logger) error {
	csv, err := os.Open(pgr.initDataPath)
	if err != nil {
		return fmt.Errorf("failed to load csv from file: %w", err)
	}
	entries := []*CPoIItem{}
	if errMarshall := gocsv.UnmarshalFile(csv, &entries); errMarshall != nil {
		return fmt.Errorf("failed to map rows to struct, %w", errMarshall)
	}
	locations := make([]*poi.PoILocation, len(entries))
	for i, v := range entries {
		d, errD := v.Domain()
		if errD != nil {
			return fmt.Errorf("failed to map test data to domain struct: %w", errD)
		}
		locations[i] = d
	}
	err = pgr.UpsertBatch(context.Background(), locations, logger)
	if err != nil {
		return fmt.Errorf("failed to perform batch upsert: %w", err)
	}
	return nil
}

func mapAvs(avs []map[string]types.AttributeValue) ([]*poi.PoILocation, error) {
	items := make([]*CPoIItem, len(avs))
	err := attributevalue.UnmarshalListOfMaps(avs, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to map list of dynamo avs: %w", err)
	}
	domain := make([]*poi.PoILocation, len(items))
	for i, v := range items {
		d, err := v.Domain()
		if err != nil {
			return nil, fmt.Errorf("failed to attribute values to domain model: %w", err)
		}
		domain[i] = d
	}
	return domain, nil
}
