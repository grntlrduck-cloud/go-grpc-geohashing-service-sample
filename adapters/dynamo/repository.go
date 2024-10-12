package dynamo

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/domain/poi"
)

var (
	TooLargeSearchAreaErr = errors.New("area too large for query")
	DBQueryErr            = errors.New("failed to query table")
)

const (
	routeHashesLimit = 100
	bboxHashesLimit  = 20
	proxHashesLimit  = 20
)

type PoIGeoRepository struct {
	dynamoClient *ClientWrapper
	logger       *zap.Logger
	tableName    string
}

func (pgr *PoIGeoRepository) UpsertBatch(
	ctx context.Context,
	pois []poi.PoILocation,
	correlationId uuid.UUID,
) error {
	return errors.New("not implemented")
}

func (pgr *PoIGeoRepository) Upsert(
	ctx context.Context,
	poi poi.PoILocation,
	correlationId uuid.UUID,
) error {
	return errors.New("not implemented")
}

func (pgr *PoIGeoRepository) GetById(
	ctx context.Context,
	id string,
	correlationId uuid.UUID,
) (poi.PoILocation, error) {
	getItemInput := &dynamodb.GetItemInput{
		TableName: aws.String(pgr.tableName),
		Key: map[string]types.AttributeValue{
			CPoIItemPK: &types.AttributeValueMemberS{Value: id},
		},
	}
	output, err := pgr.dynamoClient.GetItem(ctx, getItemInput)
	if err != nil {
		pgr.logger.Error("failed to GetItem",
			zap.String("poi_id", id),
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return poi.PoILocation{}, err
	}
	item := new(CPoIItem)
	err = attributevalue.UnmarshalMap(output.Item, item)
	if err != nil {
		pgr.logger.Error("failed to unmarshal GetItem output",
			zap.String("poi_id", id),
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return poi.PoILocation{}, err
	}
	return item.toDomain(), nil
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
		return nil, TooLargeSearchAreaErr
	}
	res, err := pgr.parallelQueryHashes(ctx, correlationId, hashes)
	if err != nil {
		pgr.logger.Error("failed to query by proximity",
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return nil, DBQueryErr
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
		return nil, TooLargeSearchAreaErr
	}
	res, err := pgr.parallelQueryHashes(ctx, correlationId, hashes)
	if err != nil {
		pgr.logger.Error("failed to query by bbox",
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return nil, DBQueryErr
	}
	return res, nil
}

func (pgr *PoIGeoRepository) GetByRoute(
	ctx context.Context,
	path []poi.Coordinates,
	correlationId uuid.UUID,
) ([]poi.PoILocation, error) {
	hashes := newHashesFromRoute(path)
	// googl s2 does not guarantee that the set MaxCells can be fulfilled
	// an arbitrary large list of hashes might be returned
	if len(hashes) > routeHashesLimit {
		pgr.logger.Error("too many hashes calculated for route",
			zap.String("correlation_id", correlationId.String()),
			zap.Int("num_hashes", len(hashes)),
		)
		return nil, TooLargeSearchAreaErr
	}
	res, err := pgr.parallelQueryHashes(ctx, correlationId, hashes)
	if err != nil {
		pgr.logger.Error("failed to query by route",
			zap.String("correlation_id", correlationId.String()),
			zap.Error(err),
		)
		return nil, DBQueryErr
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

func mapAvs(avs []map[string]types.AttributeValue) ([]poi.PoILocation, error) {
	items := new([]CPoIItem)
	err := attributevalue.UnmarshalListOfMaps(avs, items)
	if err != nil {
		return nil, fmt.Errorf("failed to map list of dynamo avs: %w", err)
	}
	domain := make([]poi.PoILocation, len(*items))
	for i, v := range *items {
		domain[i] = v.toDomain()
	}
	return domain, nil
}
