package poi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

type LocationService struct {
	repo Repository
}

func NewLocationService(repo Repository) *LocationService {
	return &LocationService{
		repo: repo,
	}
}

func (ls *LocationService) Info(
	ctx context.Context,
	id ksuid.KSUID,
	logger *zap.Logger,
) (PoILocation, error) {
	// handle context cancellation
	if ctx.Err() != nil {
		return PoILocation{}, ctx.Err()
	}
	// ensure db queries are canceled before causing sever loss of responsiveness
	logger.Debug("getting poi from db", zap.String("operation", "GetById"))
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	location, err := ls.repo.GetById(ctx, id, logger)
	if errors.Is(err, LocationNotFound) {
		logger.Warn("location not found")
	}
	return location, err
}

func (ls *LocationService) Proximity(
	ctx context.Context,
	cntr Coordinates,
	radius float64,
	logger *zap.Logger,
) ([]PoILocation, error) {
	// handle context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	logger.Info(
		"getting locations within proxmity from DB",
		zap.String("operation", "GetByProximity"),
	)
	// ensure db queries are canceled before causing sever loss of responsiveness
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	locations, err := ls.repo.GetByProximity(
		ctx,
		cntr,
		radius,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed proximity search center.lat=%f, center.lon=%f, radius_meters=%f: %w",
			cntr.Latitude,
			cntr.Longitude,
			radius,
			err,
		)
	}
	return locations, nil
}

func (ls *LocationService) Bbox(
	ctx context.Context,
	sw, ne Coordinates,
	logger *zap.Logger,
) ([]PoILocation, error) {
	// handle context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	logger.Debug(
		"getting locations in bbox from db",
		zap.String("operation", "GetByBbox"),
	)
	// ensure db queries are canceled before causing sever loss of responsiveness
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	locations, err := ls.repo.GetByBbox(ctx, sw, ne, logger)
	if err != nil {
		return nil, fmt.Errorf(
			"failed bbox search for area ne.lat=%f, ne.lon=%f, sw.lat=%f, sw.lon=%f: %w",
			ne.Latitude,
			ne.Longitude,
			sw.Latitude,
			sw.Longitude,
			err,
		)
	}
	return locations, nil
}

func (ls *LocationService) Route(
	ctx context.Context,
	wgsPath []Coordinates,
	logger *zap.Logger,
) ([]PoILocation, error) {
	// handle context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	logger.Debug(
		"getting locations along route from db",
		zap.String("operation", "GetByRoute"),
	)
	// ensure db queries are canceled before causing sever loss of responsiveness
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	locations, err := ls.repo.GetByRoute(ctx, wgsPath, logger)
	if err != nil {
		return nil, fmt.Errorf("route search failed route_length=%d: %w", len(wgsPath), err)
	}
	return locations, nil
}
