package dynamo

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/domain/poi"
)

type PoIGeoRepository struct {
	dynamoClient *ClientWrapper
	logger       *zap.Logger
	tableName    string
}

func (pgr *PoIGeoRepository) UpsertBatch(ctx context.Context, pois []poi.PoILocation) error {
	return errors.New("not implemented")
}

func (pgr *PoIGeoRepository) Upsert(ctx context.Context, poi poi.PoILocation) error {
	return errors.New("not implemented")
}

func (pgr *PoIGeoRepository) GetById(ctx context.Context, id string) (poi.PoILocation, error) {
	getItemInput := &dynamodb.GetItemInput{
		TableName: aws.String(pgr.tableName),
		Key: map[string]types.AttributeValue{
			CPoIItemPK: &types.AttributeValueMemberS{Value: id},
		},
	}
	output, getItemErr := pgr.dynamoClient.GetItem(ctx, getItemInput)
	if getItemErr != nil {
		pgr.logger.Error("failed to GetItem",
			zap.String("poi_id", id),	
			zap.Error(getItemErr),
		)
		return poi.PoILocation{}, getItemErr
	}
	item := new(CPoIItem)
	unmarshalErr := attributevalue.UnmarshalMap(output.Item, item)
	if unmarshalErr != nil {
		pgr.logger.Error("failed to unmarshal GetItem output",
			zap.String("poi_id", id),
			zap.Error(unmarshalErr),
		)
		return poi.PoILocation{}, unmarshalErr
	}
	return item.toDomain(), nil
}

func (pgr *PoIGeoRepository) GetByProximity(ctx context.Context, cntr poi.Coordinates, radius float64) ([]poi.PoILocation, error) {
 return []poi.PoILocation{}, errors.New("not implemented")
}

func (pgr *PoIGeoRepository) GetByBbox(ctx context.Context, path []poi.Coordinates, radius float64) ([]poi.PoILocation, error) {
  return []poi.PoILocation{}, errors.New("not implemented")
}

func (pgr *PoIGeoRepository) GetByRoute(ctx context.Context, path []poi.Coordinates, radius float64) ([]poi.PoILocation, error) {
  return []poi.PoILocation{}, errors.New("not implemented")
}
