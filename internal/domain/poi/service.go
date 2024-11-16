package poi

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

const proximitSearchRadiusMeters float64 = 50_000.0

type LocationService struct {
	repo   Repository
	logger *zap.Logger
}

func NewLocationService(repo Repository, logger *zap.Logger) *LocationService {
	return &LocationService{
		repo:   repo,
		logger: logger,
	}
}

func (ls *LocationService) Info(
	ctx context.Context,
	id ksuid.KSUID,
	correlationId uuid.UUID,
) (PoILocation, error) {
	ls.logger.Info(
		"getting PoI",
		zap.String("poi_id", id.String()),
		zap.String("correlation_id", correlationId.String()),
	)
	location, err := ls.repo.GetById(ctx, id, correlationId)
	if errors.Is(err, LocationNotFound) {
		ls.logger.Warn("location not found",
			zap.String("poi_id", id.String()),
			zap.String("correlation_id", correlationId.String()),
		)
		location.Id = ksuid.Max
		return location, nil
	}
	return location, err
}

func (ls *LocationService) Proximity(
	ctx context.Context,
	cntr Coordinates,
	correlationId uuid.UUID,
) ([]PoILocation, error) {
	ls.logger.Info(
		"getting PoIs in Proximity",
		zap.Float64("lon", cntr.Longitude),
		zap.Float64("lat", cntr.Latitude),
		zap.String("correlation_id", correlationId.String()),
	)
	locations, err := ls.repo.GetByProximity(
		ctx,
		cntr,
		proximitSearchRadiusMeters,
		correlationId,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to fulfill proximity query: %w", err)
	}
	return locations, nil
}

func (ls *LocationService) Bbox(
	ctx context.Context,
	sw, ne Coordinates,
	correlationId uuid.UUID,
) ([]PoILocation, error) {
	ls.logger.Info(
		"getting PoIs in bbox",
		zap.Float64("sw_lat", sw.Latitude),
		zap.Float64("sw_lon", sw.Longitude),
		zap.Float64("ne_lat", ne.Latitude),
		zap.Float64("ne_lon", ne.Longitude),
		zap.String("correlation_id", correlationId.String()),
	)
	locations, err := ls.repo.GetByBbox(ctx, sw, ne, correlationId)
	if err != nil {
		return nil, fmt.Errorf("unable to fulfill bbox search for area: %w", err)
	}
	return locations, nil
}

func (ls *LocationService) Route(
	ctx context.Context,
	wgsPath []Coordinates,
	correlationId uuid.UUID,
) ([]PoILocation, error) {
	ls.logger.Info(
		"getting PoIs along route",
		zap.Int("num_coordinates", len(wgsPath)),
		zap.String("correlation_id", correlationId.String()),
	)
	locations, err := ls.repo.GetByRoute(ctx, wgsPath, correlationId)
	if err != nil {
		return nil, fmt.Errorf("unable to fulfill route search: %w", err)
	}
	return locations, nil
}
