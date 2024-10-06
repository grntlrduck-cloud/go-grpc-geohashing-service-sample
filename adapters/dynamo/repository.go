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

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/domain/poi"
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
	pgr.logger.Info(
		fmt.Sprintf("calculated #%d geo hashes", len(hashes)),
		zap.String("correlation_id", correlationId.String()),
	)
	pgr.logger.Info(
		fmt.Sprintf("first hash has min=%d and max=%d", hashes[0].min(), hashes[0].max()),
		zap.String("correlation_id", correlationId.String()),
	)
	return []poi.PoILocation{}, errors.New("not implemented")
}

func (pgr *PoIGeoRepository) GetByBbox(
	ctx context.Context,
	sw, ne poi.Coordinates,
	correlationId uuid.UUID,
) ([]poi.PoILocation, error) {
	hashes := newHashesFromBbox(ne, sw)
	pgr.logger.Info(
		fmt.Sprintf("calculated #%d geo hashes", len(hashes)),
		zap.String("correlation_id", correlationId.String()),
	)
	return []poi.PoILocation{}, errors.New("not implemented")
}

func (pgr *PoIGeoRepository) GetByRoute(
	ctx context.Context,
	path []poi.Coordinates,
	correlationId uuid.UUID,
) ([]poi.PoILocation, error) {
	hashes := newHashesFromRoute(path)
	queries := pgr.queryInputFromHashes(hashes)
	// TODO: concurrently fetch for each query, handle errors, and merge results by using Wait and ErrorGroups or channels
	pois, err := pgr.handleQuery(ctx, queries[0])
	if err != nil {
		return nil, fmt.Errorf("failed to query pois for route: %w", err)
	}
	pgr.logger.Info(
		fmt.Sprintf("calculated #%d geo hashes and created #%d queries", len(hashes), len(queries)),
		zap.String("correlation_id", correlationId.String()),
	)
	return pois, errors.New("not implemented")
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

func (pgr *PoIGeoRepository) handleQuery(
	ctx context.Context,
	input *dynamodb.QueryInput,
) ([]poi.PoILocation, error) {
	items := make([]map[string]types.AttributeValue, 0)
	queryResult, err := pgr.dynamoClient.QueryItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query PoIs: %w", err)
	}
	items = append(items, queryResult.Items...)
	for queryResult.LastEvaluatedKey != nil {
		input.ExclusiveStartKey = queryResult.LastEvaluatedKey
		res, err := pgr.dynamoClient.QueryItem(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to call query page: %w", err)
		}
		items = append(items, res.Items...)
	}
	domain, err := mapAvs(items)
	if err != nil {
		return nil, fmt.Errorf("failed to map results: %w", err)
	}
	return domain, nil
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
