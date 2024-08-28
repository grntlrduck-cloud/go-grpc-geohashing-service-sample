package poi

import "context"

type Repository interface {
	UpsertBatch(ctx context.Context, pois []PoILocation) error
	Upsert(ctx context.Context, poi PoILocation) error
	GetById(ctx context.Context, id string) (PoILocation, error)
	GetByProximity(ctx context.Context, cntr Coordinates, radius float64) ([]PoILocation, error)
	GetByBbox(ctx context.Context, sw, ne Coordinates) ([]PoILocation, error)
	BetByRoute(ctx context.Context, path []Coordinates, radius float64) ([]PoILocation, error)
}
