package poi

import (
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TODO: repository usage
type LocationService struct {
	repo   Repository // nolint:unused
	logger *zap.Logger
}

func (ls *LocationService) Info(id string, correlationId uuid.UUID) (PoILocation, error) {
	ls.logger.Info(
		"getting PoI",
		zap.String("poi_id", id),
		zap.String("correlation_id", correlationId.String()),
	)
	return PoILocation{}, nil
}

func (ls *LocationService) Proximity(
	cntr Coordinates,
	correlationId uuid.UUID,
) ([]PoILocation, error) {
	ls.logger.Info(
		"getting PoIs in Proximity",
		zap.Float64("lon", cntr.Longitude),
		zap.Float64("lat", cntr.Latitude),
		zap.String("correlation_id", correlationId.String()),
	)
	return []PoILocation{}, nil
}

func (ls *LocationService) Bbox(
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
	return []PoILocation{}, nil
}

func (ls *LocationService) Route(
	wgsPath []Coordinates,
	correlationId uuid.UUID,
) ([]PoILocation, error) {
	ls.logger.Info(
		"getting PoIs along route",
		zap.Int("num_coordinates", len(wgsPath)),
		zap.String("correlation_id", correlationId.String()),
	)
	return []PoILocation{}, nil
}
