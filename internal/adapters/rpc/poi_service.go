package rpc

import (
	"context"
	"errors"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"

	poi_v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/poi"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/domain/poi"
)

var (
	missingOrInvalidCorrelationIdStatus = status.Errorf(
		codes.InvalidArgument,
		"invalid request, X-Correlation-Id header is required in UUID v4 format",
	)
	invalidGeoParamsStatus = status.Errorf(
		codes.InvalidArgument,
		"invalid geo search arguments, ensure correct coordinates and the number of parameters required",
	)

	severErrStatus = status.Errorf(codes.Internal, "server error, failed to process request")
)

const (
	minSearchRadiusMeters float64 = 1000.0    // 1 km
	maxSearchRadiusMeters float64 = 100_000.0 // 100 km
)

type PoIRpcService struct {
	poi_v1.UnimplementedPoIServiceServer
	logger          *zap.Logger
	locationService *poi.LocationService
}

func NewPoIRpcService(logger *zap.Logger, locationService *poi.LocationService) *PoIRpcService {
	return &PoIRpcService{
		logger:          logger,
		locationService: locationService,
	}
}

func (p *PoIRpcService) PoI(
	ctx context.Context,
	request *poi_v1.PoIRequest,
) (*poi_v1.PoIResponse, error) {
	// validate request
	if request == nil || request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "missing argument: id")
	}
	correlationId, err := getCorrelationId(ctx)
	if err != nil {
		return nil, missingOrInvalidCorrelationIdStatus
	}

	// setting response header for client side tracing
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationId.String()))

	kId, err := ksuid.Parse(request.Id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid location id format: given id=%s",
			request.Id,
		)
	}

	// set correlationId and PoI ID for logger
	logger := p.logger.With(
		zap.String("correlation_id", correlationId.String()),
		zap.String("rpc_method", "PoI"),
		zap.String("id", kId.String()),
	)
	logger.Info(
		"processing PoI rpc",
	)

	// process request
	location, err := p.locationService.Info(ctx, kId, logger)

	// handle errors accordingly
	if errors.Is(err, poi.LocationNotFound) {
		return nil, status.Errorf(codes.NotFound, "location not found: id=%s", request.Id)
	}
	if err != nil {
		logger.Error("failed to get poi by id", zap.Error(err))
		return nil, severErrStatus
	}

	response := poi_v1.PoIResponse{
		Poi: poiToProto(location),
	}
	p.logger.Info(
		"returning response for PoI rpc",
	)
	return &response, nil
}

func (p *PoIRpcService) Proximity(
	ctx context.Context,
	request *poi_v1.ProximityRequest,
) (*poi_v1.PoISearchResponse, error) {
	// validate request
	if request == nil || request.Center == nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"center must be given to perform proximity search",
		)
	}
	if request.RadiusMeters < minSearchRadiusMeters ||
		request.RadiusMeters > maxSearchRadiusMeters {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid radius: radius=%f must be between 1000 m and 200_000 m",
			request.RadiusMeters,
		)
	}
	correlationId, err := getCorrelationId(ctx)
	if err != nil {
		return nil, missingOrInvalidCorrelationIdStatus
	}

	// setting response header for client side tracing
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationId.String()))

	// set correlationId for logging
	logger := p.logger.With(
		zap.String("correlation_id", correlationId.String()),
		zap.String("rpc_method", "Proximity"),
		zap.Float64("lat", request.Center.Lat),
		zap.Float64("lon", request.Center.Lon),
	)
	logger.Info(
		"processing Proximity rpc",
	)

	// process request
	cntr := poi.Coordinates{
		Latitude:  request.Center.Lat,
		Longitude: request.Center.Lon,
	}
	locations, err := p.locationService.Proximity(ctx, cntr, request.RadiusMeters, logger)

	// handle errors accordingly
	if errors.Is(err, poi.InvalidSearchCoordinatesErr) {
		return nil, invalidGeoParamsStatus
	}
	if err != nil {
		logger.Error("unable to handle request", zap.Error(err))
		return nil, severErrStatus
	}

	// log and return
	logger.Info(
		"returning response for Proximity RPC",
		zap.Int("num_locations", len(locations)),
	)
	resp := buildPoISearchResponse(locations)
	return resp, nil
}

func (p *PoIRpcService) BBox(
	ctx context.Context,
	request *poi_v1.BBoxRequest,
) (*poi_v1.PoISearchResponse, error) {
	// validate the bbox request
	if request == nil || request.Bbox == nil || request.Bbox.Sw == nil || request.Bbox.Ne == nil {
		return nil, status.Errorf(codes.InvalidArgument, "bounding box coordinates must be defined")
	}
	correlationId, err := getCorrelationId(ctx)
	if err != nil {
		return nil, missingOrInvalidCorrelationIdStatus
	}

	// setting response header for client side tracing
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationId.String()))

	// set correlationId for logging
	logger := p.logger.With(
		zap.String("correlation_id", correlationId.String()),
		zap.String("rpc_method", "Bbox"),
		zap.Float64("ne.lat", request.Bbox.Ne.Lat),
		zap.Float64("ne.lon", request.Bbox.Ne.Lon),
		zap.Float64("sw.lat", request.Bbox.Sw.Lat),
		zap.Float64("sw.lon", request.Bbox.Sw.Lon),
	)
	logger.Info(
		"processing BBox rpc",
	)

	// process request
	sw := poi.Coordinates{
		Latitude:  request.Bbox.Sw.Lat,
		Longitude: request.Bbox.Sw.Lon,
	}
	ne := poi.Coordinates{
		Latitude:  request.Bbox.Ne.Lat,
		Longitude: request.Bbox.Ne.Lon,
	}
	locations, err := p.locationService.Bbox(ctx, sw, ne, logger)

	// handle errors accordingly
	if errors.Is(err, poi.InvalidSearchCoordinatesErr) {
		return nil, invalidGeoParamsStatus
	}
	if err != nil {
		logger.Error("unable to handle request", zap.Error(err))
		return nil, severErrStatus
	}

	// log and return
	logger.Info(
		"returning response for BBox RPC",
		zap.Int("num_locations", len(locations)),
	)
	resp := buildPoISearchResponse(locations)
	return resp, nil
}

func (p *PoIRpcService) Route(
	ctx context.Context,
	request *poi_v1.RouteRequest,
) (*poi_v1.PoISearchResponse, error) {
	// validate request
	if request == nil || request.Route == nil || len(request.Route) < 2 ||
		len(request.Route) > 100 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"a route must at least have two coordinates and not more then 100",
		)
	}
	correlationId, err := getCorrelationId(ctx)
	if err != nil {
		return nil, missingOrInvalidCorrelationIdStatus
	}

	// setting response header for client side tracing
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationId.String()))

	// set correlatioId for logging
	logger := p.logger.With(
		zap.String("correlation_id", correlationId.String()),
		zap.String("rpc_method", "Route"),
		zap.Int("num_route_points", len(request.Route)),
	)
	logger.Info(
		"processing Route rpc",
	)

	// process request
	path := coordinatesPathFromProto(request.Route)
	locations, err := p.locationService.Route(ctx, path, logger)

	// handle the errors accordingly
	if errors.Is(err, poi.InvalidSearchCoordinatesErr) {
		return nil, invalidGeoParamsStatus
	}
	if err != nil {
		logger.Error("unable to handle request", zap.Error(err))
		return nil, severErrStatus
	}

	// log and return
	logger.Info(
		"returning response for Route RPC",
		zap.Int("num_locations", len(locations)),
	)
	resp := buildPoISearchResponse(locations)
	return resp, nil
}

func (p *PoIRpcService) Register(server *grpc.Server) {
	poi_v1.RegisterPoIServiceServer(server, p)
}

func (p *PoIRpcService) RegisterProxy(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return poi_v1.RegisterPoIServiceHandlerFromEndpoint(
		ctx,
		mux,
		endpoint,
		opts,
	)
}

func buildPoISearchResponse(l []poi.PoILocation) *poi_v1.PoISearchResponse {
	proto := locationsToProto(l)
	return &poi_v1.PoISearchResponse{
		Items: proto,
	}
}

func locationsToProto(l []poi.PoILocation) []*poi_v1.PoI {
	pois := make([]*poi_v1.PoI, len(l))
	for i, v := range l {
		pois[i] = poiToProto(v)
	}
	return pois
}

func poiToProto(p poi.PoILocation) *poi_v1.PoI {
	return &poi_v1.PoI{
		Id: p.Id.String(),
		Coordinate: &poi_v1.Coordinate{
			Lat: p.Location.Latitude,
			Lon: p.Location.Longitude,
		},
		Address: &poi_v1.Address{
			Street:       p.Address.Street,
			StreetNumber: p.Address.StreetNumber,
			ZipCode:      p.Address.ZipCode,
			City:         p.Address.City,
			Country:      p.Address.CountryCode,
		},
		Features: p.Features,
	}
}

func coordinatesPathFromProto(c []*poi_v1.Coordinate) []poi.Coordinates {
	path := make([]poi.Coordinates, len(c))
	for i, v := range c {
		path[i] = poi.Coordinates{
			Latitude:  v.Lat,
			Longitude: v.Lon,
		}
	}
	return path
}
