package dynamo

import (
	"context"
	"errors"
	"fmt"

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
	pgr.logger.Info(fmt.Sprintf("calculated #%d geo hashes", len(hashes)))
	pgr.logger.Info(
		fmt.Sprintf("first hash has min=%d and max=%d", hashes[0].min(), hashes[0].max()),
	)
	return []poi.PoILocation{}, errors.New("not implemented")
}

func (pgr *PoIGeoRepository) GetByBbox(
	ctx context.Context,
	sw, ne poi.Coordinates,
	correlationId uuid.UUID,
) ([]poi.PoILocation, error) {
	hashes := newHashesFromBbox(ne, sw)
	pgr.logger.Info(fmt.Sprintf("calculated #%d geo hashes", len(hashes)))
	return []poi.PoILocation{}, errors.New("not implemented")
}

func (pgr *PoIGeoRepository) GetByRoute(
	ctx context.Context,
	path []poi.Coordinates,
	correlationId uuid.UUID,
) ([]poi.PoILocation, error) {
	hashes := newHashesFromRoute(path)
	pgr.logger.Info(fmt.Sprintf("calculated #%d geo hashes", len(hashes)))
	return []poi.PoILocation{}, errors.New("not implemented")
}
