package poi

import (
	"context"
	"errors"
	"fmt"

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
	logger.Info("getting poi from db")
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
	logger.Info(
		"getting locations within proxmity from DB",
	)
	locations, err := ls.repo.GetByProximity(
		ctx,
		cntr,
		radius,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to fulfill proximity query: %w", err)
	}
	return locations, nil
}

func (ls *LocationService) Bbox(
	ctx context.Context,
	sw, ne Coordinates,
	logger *zap.Logger,
) ([]PoILocation, error) {
	logger.Info(
		"getting locations in bbox from db",
	)
	locations, err := ls.repo.GetByBbox(ctx, sw, ne, logger)
	if err != nil {
		return nil, fmt.Errorf("unable to fulfill bbox search for area: %w", err)
	}
	return locations, nil
}

func (ls *LocationService) Route(
	ctx context.Context,
	wgsPath []Coordinates,
	logger *zap.Logger,
) ([]PoILocation, error) {
	logger.Info(
		"getting locations along route from db",
	)
	locations, err := ls.repo.GetByRoute(ctx, wgsPath, logger)
	if err != nil {
		return nil, fmt.Errorf("unable to fulfill route search: %w", err)
	}
	return locations, nil
}
