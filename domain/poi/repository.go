package poi

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	UpsertBatch(ctx context.Context, pois []PoILocation, correlationId uuid.UUID) error

	Upsert(ctx context.Context, poi PoILocation, correlationId uuid.UUID) error

	GetById(ctx context.Context, id string, correlationId uuid.UUID) (PoILocation, error)

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
