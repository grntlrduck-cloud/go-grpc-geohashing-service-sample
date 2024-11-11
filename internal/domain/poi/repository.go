package poi

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/segmentio/ksuid"
)

var (
	TooLargeSearchAreaErr = errors.New("area too large for query")
	DBQueryErr            = errors.New("failed to query table")
	LocationNotFound      = errors.New("location does not exist")
	DBEntityMappingErr    = errors.New("failed to map too location entity")
	DBUpsertErr           = errors.New("failed to upsert entity")
	DBBatchUpsertErr      = errors.New("failed to upsert batch")
)

type Repository interface {
	UpsertBatch(ctx context.Context, pois []PoILocation, correlationId uuid.UUID) error

	Upsert(ctx context.Context, domain PoILocation, correlationId uuid.UUID) error

	GetById(ctx context.Context, id ksuid.KSUID, correlationId uuid.UUID) (PoILocation, error)

	GetByProximity(
		ctx context.Context,
		cntr Coordinates,
		radius float64,
		correlationId uuid.UUID,
	) ([]PoILocation, error)

	GetByBbox(
		ctx context.Context,
		sw, ne Coordinates,
		correlationId uuid.UUID,
	) ([]PoILocation, error)

	GetByRoute(
		ctx context.Context,
		path []Coordinates,
		correlationId uuid.UUID,
	) ([]PoILocation, error)
}
