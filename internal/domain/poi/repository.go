package poi

import (
	"context"
	"errors"

	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

var (
	TooLargeSearchAreaErr       = errors.New("area too large for query")
	DBQueryErr                  = errors.New("failed to query table")
	LocationNotFound            = errors.New("location does not exist")
	DBEntityMappingErr          = errors.New("failed to map too location entity")
	DBUpsertErr                 = errors.New("failed to upsert entity")
	DBBatchUpsertErr            = errors.New("failed to upsert batch")
	InvalidSearchCoordinatesErr = errors.New("invalid geo search parameters: invalid coordinates")
)

type Repository interface {
	UpsertBatch(ctx context.Context, pois []PoILocation, logger *zap.Logger) error

	Upsert(ctx context.Context, domain PoILocation, logger *zap.Logger) error

	GetById(ctx context.Context, id ksuid.KSUID, logger *zap.Logger) (PoILocation, error)

	GetByProximity(
		ctx context.Context,
		cntr Coordinates,
		radius float64,
		logger *zap.Logger,
	) ([]PoILocation, error)

	GetByBbox(
		ctx context.Context,
		sw, ne Coordinates,
		logger *zap.Logger,
	) ([]PoILocation, error)

	GetByRoute(
		ctx context.Context,
		path []Coordinates,
		logger *zap.Logger,
	) ([]PoILocation, error)
}
