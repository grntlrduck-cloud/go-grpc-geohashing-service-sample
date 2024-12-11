package poi

import (
	"context"
	"errors"

	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

var (
	ErrTooLargeSearchArea       = errors.New("area too large for query")
	ErrDBQuery                  = errors.New("failed to query table")
	ErrLocationNotFound         = errors.New("location does not exist")
	ErrDBEntityMapping          = errors.New("failed to map too location entity")
	ErrDBUpsert                 = errors.New("failed to upsert entity")
	ErrDBBatchUpsert            = errors.New("failed to upsert batch")
	ErrInvalidSearchCoordinates = errors.New("invalid geo search parameters: invalid coordinates")
)

type Repository interface {
	UpsertBatch(ctx context.Context, pois []*PoILocation, logger *zap.Logger) error

	Upsert(ctx context.Context, domain *PoILocation, logger *zap.Logger) error

	GetByID(ctx context.Context, id ksuid.KSUID, logger *zap.Logger) (*PoILocation, error)

	GetByProximity(
		ctx context.Context,
		cntr Coordinates,
		radius float64,
		logger *zap.Logger,
	) ([]*PoILocation, error)

	GetByBbox(
		ctx context.Context,
		sw, ne Coordinates,
		logger *zap.Logger,
	) ([]*PoILocation, error)

	GetByRoute(
		ctx context.Context,
		path []Coordinates,
		logger *zap.Logger,
	) ([]*PoILocation, error)
}
