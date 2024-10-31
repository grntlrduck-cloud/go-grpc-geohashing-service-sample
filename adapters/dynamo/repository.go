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
	"github.com/google/uuid"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/app"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/domain/poi"
)

const (
	routeHashesLimit   = 100
	bboxHashesLimit    = 20
	proxHashesLimit    = 20
	dynamoMaxBatchSize = 25
	testInitDataPath   = "config/db/local/cpoi_dynamo_items_int_test.csv"
)

type PoIGeoRepository struct {
	dynamoClient    *ClientWrapper
	logger          *zap.Logger
	tableName       string
	createInitTable bool
}

type PoIGeoRepositoryOptions func(p *PoIGeoRepository)

func WithDynamoClientWrapper(client *ClientWrapper) PoIGeoRepositoryOptions {
	return func(p *PoIGeoRepository) {
		p.dynamoClient = client
	}
}

func WithLogger(logger *zap.Logger) PoIGeoRepositoryOptions {
	return func(p *PoIGeoRepository) {
		p.logger = logger
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

func NewPoIGeoRepository(opts ...PoIGeoRepositoryOptions) (*PoIGeoRepository, error) {
	repo := &PoIGeoRepository{
		tableName: "NOT_DEFINED",
	}
	for _, opt := range opts {
		opt(repo)
	}
	if repo.logger == nil {
		repo.logger = app.NewDevLogger()
	}
	if repo.dynamoClient == nil {
		cl, err := NewClientWrapper()
		if err != nil {
			return nil, fmt.Errorf("dyanmo client was nil but failed to initialize: %w", err)
		}
		repo.dynamoClient = cl
	}
	if repo.createInitTable {
		err := repo.createTableAndLoadData()
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
	pois []poi.PoILocation,
	correlationId uuid.UUID,
) error {
	// map domain model to dynamo items and marshall to AttributeValues
	// and assemble list of WriteRequests
	rqsts := make([]types.WriteRequest, len(pois))
	for i, v := range pois {
		item := newItemFromDomain(v)
		av, err := attributevalue.MarshalMap(&item)
		if err != nil {
			return poi.DBEntityMappingErr
		}
		rqsts[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: av}}
	}
	numChunks := (len(pois) + dynamoMaxBatchSize - 1) / dynamoMaxBatchSize
	chunks := make([][]types.WriteRequest, numChunks)

	// create chunks with max 25 items
	for dynamoMaxBatchSize < len(rqsts) {
		rqsts, chunks = rqsts[dynamoMaxBatchSize:], append(
			chunks,
			rqsts[0:dynamoMaxBatchSize:dynamoMaxBatchSize],
		)
	}
	chunks = append(chunks, rqsts)

	// upsert chunks
	var errs []error
	for _, c := range chunks {
		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{pgr.tableName: c},
		}
		_, err := pgr.dynamoClient.BatchPutItem(ctx, input)
		if err != nil {
			// just  retry once after 10 ms sleep
			// don't do that in real world production scenario
			// implement exponential backoff instead
			pgr.logger.Warn(
				"retrying batch PutItem",
				zap.String("correlation_id", correlationId.String()),
			)
			time.Sleep(time.Millisecond * 10)
			_, err := pgr.dynamoClient.BatchPutItem(ctx, input)
			if err != nil {
				pgr.logger.Error(
					"failed to perform batch PutItem",
					zap.String("correlation_id", correlationId.String()),
				)
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		pgr.logger.Error("batch upsert incomplete",
			zap.Error(errs[0]),
			zap.String("correlation_id", correlationId.String()),
			zap.Int("num_failed_batches", len(errs)),
		)
		return poi.DBBatchUpsertErr
	}
	return nil
}

func (pgr *PoIGeoRepository) Upsert(
	ctx context.Context,
	domain poi.PoILocation,
	correlationId uuid.UUID,
) error {
	item := newItemFromDomain(domain)
	avs, err := attributevalue.MarshalMap(&item)
	if err != nil {
		pgr.logger.Error("unable to marshall item to DynamoDB AttributeValues",
			zap.Error(err),
			zap.String("correlation_id", correlationId.String()),
		)
		return poi.DBEntityMappingErr
	}
	putItemInput := &dynamodb.PutItemInput{
		Item:      avs,
		TableName: &pgr.tableName,
	}
	_, err = pgr.dynamoClient.PutItem(ctx, putItemInput)
	if err != nil {
		pgr.logger.Error("failed to PutItem",
			zap.Error(err),
			zap.String("correlation_id", correlationId.String()),
		)
		return poi.DBUpsertErr
	}
	return nil
}

func (pgr *PoIGeoRepository) GetById(
	ctx context.Context,
	id ksuid.KSUID,
	correlationId uuid.UUID,
) (poi.PoILocation, error) {
	getItemInput := &dynamodb.GetItemInput{
		TableName: aws.String(pgr.tableName),
		Key: map[string]types.AttributeValue{
			CPoIItemPK: &types.AttributeValueMemberS{Value: id.String()},
		},
	}
	output, err := pgr.dynamoClient.GetItem(ctx, getItemInput)
	if err != nil {
		pgr.logger.Error("failed to GetItem",
			zap.String("poi_id", id.String()),
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return poi.PoILocation{}, err
	}
	item := new(CPoIItem)
	err = attributevalue.UnmarshalMap(output.Item, item)
	if err != nil {
		pgr.logger.Error("failed to unmarshal GetItem output",
			zap.String("poi_id", id.String()),
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return poi.PoILocation{}, err
	}
	if item.Pk == "" {
		return poi.PoILocation{}, poi.LocationNotFound
	}
	return item.Domain()
}

func (pgr *PoIGeoRepository) GetByProximity(
	ctx context.Context,
	cntr poi.Coordinates,
	radius float64,
	correlationId uuid.UUID,
) ([]poi.PoILocation, error) {
	hashes := newHashesFromRadiusCenter(cntr, radius)
	if len(hashes) > proxHashesLimit {
		pgr.logger.Error("too many hashes calculated for proximity",
			zap.String("correlation_id", correlationId.String()),
			zap.Int("num_hashes", len(hashes)),
		)
		return nil, poi.TooLargeSearchAreaErr
	}
	res, err := pgr.parallelQueryHashes(ctx, correlationId, hashes)
	if err != nil {
		pgr.logger.Error("failed to query by proximity",
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return nil, poi.DBQueryErr
	}
	return res, nil
}

func (pgr *PoIGeoRepository) GetByBbox(
	ctx context.Context,
	sw, ne poi.Coordinates,
	correlationId uuid.UUID,
) ([]poi.PoILocation, error) {
	hashes := newHashesFromBbox(ne, sw)
	if len(hashes) > bboxHashesLimit {
		pgr.logger.Error("too many hashes calculated for bbox",
			zap.String("correlation_id", correlationId.String()),
			zap.Int("num_hashes", len(hashes)),
		)
		return nil, poi.TooLargeSearchAreaErr
	}
	res, err := pgr.parallelQueryHashes(ctx, correlationId, hashes)
	if err != nil {
		pgr.logger.Error("failed to query by bbox",
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return nil, poi.DBQueryErr
	}
	return res, nil
}

func (pgr *PoIGeoRepository) GetByRoute(
	ctx context.Context,
	path []poi.Coordinates,
	correlationId uuid.UUID,
) ([]poi.PoILocation, error) {
	hashes := newHashesFromRoute(path)
	// google s2 does not guarantee that the set MaxCells can be fulfilled
	// an arbitrary large list of hashes might be returned
	if len(hashes) > routeHashesLimit {
		pgr.logger.Error("too many hashes calculated for route",
			zap.String("correlation_id", correlationId.String()),
			zap.Int("num_hashes", len(hashes)),
		)
		return nil, poi.TooLargeSearchAreaErr
	}
	res, err := pgr.parallelQueryHashes(ctx, correlationId, hashes)
	if err != nil {
		pgr.logger.Error("failed to query by route",
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return nil, poi.DBQueryErr
	}
	return res, nil
}

func (pgr *PoIGeoRepository) parallelQueryHashes(
	ctx context.Context,
	correlationId uuid.UUID,
	hashes []geoHash,
) ([]poi.PoILocation, error) {
	queries := pgr.queryInputFromHashes(hashes)
	pgr.logger.Info("sending parallel requests for geo query",
		zap.String("correlation_id", correlationId.String()),
		zap.Int("queries", len(queries)),
	)
	resC := make(chan poiQueryResult)
	errGrp, gctx := errgroup.WithContext(ctx)
	errGrp.SetLimit(10)
	for _, v := range queries {
		errGrp.Go(func() error {
			qres := pgr.query(gctx, v)
			if qres.err != nil {
				return qres.err
			}
			select {
			case resC <- qres:
			case <-gctx.Done():
				return gctx.Err()
			}
			return nil
		})
	}
	go func() {
		_ = errGrp.Wait()
		close(resC)
	}()
	var pois []poi.PoILocation
	for r := range resC {
		pois = append(pois, r.pois...)
	}
	if err := errGrp.Wait(); err != nil {
		return nil, fmt.Errorf("one or more concurrent dynamoDb queries failed: %w", err)
	}
	return pois, nil
}

type poiQueryResult struct {
	pois []poi.PoILocation
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
		res, err := pgr.dynamoClient.QueryItem(ctx, input)
		if err != nil {
			return poiQueryResult{nil, fmt.Errorf("failed to call query page: %w", err)}
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
		println(v.trimmed(CPoIItemGeoHashKeyLength))
		query := &dynamodb.QueryInput{
			TableName:              aws.String(pgr.tableName),
			IndexName:              aws.String(CPoIItemGeoIndexName),
			KeyConditionExpression: aws.String(keyCondition),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberN{
					Value: strconv.FormatUint(v.trimmed(CPoIItemGeoHashKeyLength), 10),
				},
				":skmin": &types.AttributeValueMemberN{Value: strconv.FormatUint(v.min(), 10)},
				":skmax": &types.AttributeValueMemberN{Value: strconv.FormatUint(v.max(), 10)},
			},
		}
		queries = append(queries, query)
	}
	return queries
}

func (pgr *PoIGeoRepository) createTableAndLoadData() error {
	pgr.logger.Warn("table will be created and initialized with initial data!")
	err := pgr.createInitPoiTable()
	if err != nil {
		return fmt.Errorf("failed to initialize table for local testing: %w", err)
	}
	err = pgr.loadInitData()
	if err != nil {
		return fmt.Errorf("failed to load initial data into table for testing: %w", err)
	}
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

func (pgr *PoIGeoRepository) loadInitData() error {
	csv, err := os.Open(testInitDataPath)
	if err != nil {
		return fmt.Errorf("failed to load csv from file: %w", err)
	}
	entries := []*CPoIItem{}
	if err := gocsv.UnmarshalFile(csv, &entries); err != nil {
		return fmt.Errorf("failed to map rows to struct, %w", err)
	}
	locations := make([]poi.PoILocation, len(entries))
	for i, v := range entries {
		d, err := v.Domain()
		if err != nil {
			return fmt.Errorf("failed to map test data to domain struct: %w", err)
		}
		locations[i] = d
	}
	err = pgr.UpsertBatch(context.Background(), locations, uuid.New())
	if err != nil {
		return fmt.Errorf("failed to perform batch upsert: %w", err)
	}
	return nil
}

func mapAvs(avs []map[string]types.AttributeValue) ([]poi.PoILocation, error) {
	items := new([]CPoIItem)
	err := attributevalue.UnmarshalListOfMaps(avs, items)
	if err != nil {
		return nil, fmt.Errorf("failed to map list of dynamo avs: %w", err)
	}
	domain := make([]poi.PoILocation, len(*items))
	for i, v := range *items {
		d, err := v.Domain()
		if err != nil {
			return nil, fmt.Errorf("failed to attribute values to domain model: %w", err)
		}
		domain[i] = d
	}
	return domain, nil
}
