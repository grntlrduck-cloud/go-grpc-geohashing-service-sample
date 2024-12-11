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
	missingOrInvalidCorrelationIDStatus = status.Errorf(
		codes.InvalidArgument,
		"invalid request, X-Correlation-Id header is required in UUID v4 format",
	)
	requestCenceledStatus = status.Errorf(codes.Canceled, "client aborted connection")

	invalidGeoParamsMessage = "invalid geo search arguments, ensure correct coordinates and the number of parameters required"

	severErrMessage = "server error, failed to process request"
)

const (
	minSearchRadiusMeters float64 = 1000.0    // 1 km
	maxSearchRadiusMeters float64 = 100_000.0 // 100 km
)

type PoIRPCService struct {
	poi_v1.UnimplementedPoIServiceServer
	logger          *zap.Logger
	locationService *poi.LocationService
}

func NewPoIRPCService(logger *zap.Logger, locationService *poi.LocationService) *PoIRPCService {
	return &PoIRPCService{
		logger:          logger,
		locationService: locationService,
	}
}

func (p *PoIRPCService) PoI(
	ctx context.Context,
	request *poi_v1.PoIRequest,
) (*poi_v1.PoIResponse, error) {
	// handle context cancellation
	if ctx.Err() != nil {
		return nil, requestCenceledStatus
	}
	if request == nil || request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "missing argument: id")
	}
	correlationID, err := getCorrelationID(ctx)
	if err != nil {
		return nil, missingOrInvalidCorrelationIDStatus
	}

	// setting response header for client side tracing
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationID.String()))

	kID, err := ksuid.Parse(request.Id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid location id format: given id=%s",
			request.Id,
		)
	}

	// set correlationID and PoI ID for logger
	logger := p.logger.With(
		zap.String("correlation_id", correlationID.String()),
		zap.String("rpc_method", "PoI"),
		zap.String("location_id", kID.String()),
	)
	logger.Info(
		"processing PoI rpc",
	)

	// process request
	location, err := p.locationService.Info(ctx, kID, logger)

	// handle errors accordingly
	if errors.Is(err, poi.ErrLocationNotFound) ||
		errors.Is(errors.Unwrap(err), poi.ErrLocationNotFound) {
		return nil, status.Errorf(codes.NotFound, "location not found: id=%s", request.Id)
	}
	if err != nil {
		logger.Error("failed to get poi by id", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "%s: %v", severErrMessage, err)
	}
	response := poi_v1.PoIResponse{
		Poi: poiToProto(location),
	}
	p.logger.Info(
		"returning response for PoI rpc",
	)
	return &response, nil
}

func (p *PoIRPCService) Proximity(
	ctx context.Context,
	request *poi_v1.ProximityRequest,
) (*poi_v1.PoISearchResponse, error) {
	// handle context cancellation
	if ctx.Err() != nil {
		return nil, requestCenceledStatus
	}
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
			"invalid radius: radius=%f must be between 1000 m (1 km) and 100000 m (100 km)",
			request.RadiusMeters,
		)
	}
	correlationID, err := getCorrelationID(ctx)
	if err != nil {
		return nil, missingOrInvalidCorrelationIDStatus
	}

	// setting response header for client side tracing
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationID.String()))

	// set correlationID for logging
	logger := p.logger.With(
		zap.String("correlation_id", correlationID.String()),
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
	if errors.Is(err, poi.ErrInvalidSearchCoordinates) ||
		errors.Is(errors.Unwrap(err), poi.ErrInvalidSearchCoordinates) {
		return nil, status.Errorf(codes.InvalidArgument, "%s: %v", invalidGeoParamsMessage, err)
	}
	if err != nil {
		logger.Error("unable to handle request", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "%s: %v", severErrMessage, err)
	}

	// log and return
	logger.Info(
		"returning response for Proximity RPC",
		zap.Int("num_locations", len(locations)),
	)
	resp := buildPoISearchResponse(locations)
	return resp, nil
}

func (p *PoIRPCService) BBox(
	ctx context.Context,
	request *poi_v1.BBoxRequest,
) (*poi_v1.PoISearchResponse, error) {
	// handle context cancellation
	if ctx.Err() != nil {
		return nil, requestCenceledStatus
	}
	// validate the bbox request
	if request == nil || request.Bbox == nil || request.Bbox.Sw == nil || request.Bbox.Ne == nil {
		return nil, status.Errorf(codes.InvalidArgument, "bounding box coordinates must be defined")
	}
	correlationID, err := getCorrelationID(ctx)
	if err != nil {
		return nil, missingOrInvalidCorrelationIDStatus
	}

	// setting response header for client side tracing
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationID.String()))

	// set correlationID for logging
	logger := p.logger.With(
		zap.String("correlation_id", correlationID.String()),
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
	if errors.Is(err, poi.ErrInvalidSearchCoordinates) ||
		errors.Is(errors.Unwrap(err), poi.ErrInvalidSearchCoordinates) {
		return nil, status.Errorf(codes.InvalidArgument, "%s: %v", invalidGeoParamsMessage, err)
	}
	if err != nil {
		logger.Error("unable to handle request", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "%s: %v", severErrMessage, err)
	}

	// log and return
	logger.Info(
		"returning response for BBox RPC",
		zap.Int("num_locations", len(locations)),
	)
	resp := buildPoISearchResponse(locations)
	return resp, nil
}

func (p *PoIRPCService) Route(
	ctx context.Context,
	request *poi_v1.RouteRequest,
) (*poi_v1.PoISearchResponse, error) {
	// handle context cancellation
	if ctx.Err() != nil {
		return nil, requestCenceledStatus
	}
	// validate request
	if request == nil || request.Route == nil || len(request.Route) < 2 ||
		len(request.Route) > 100 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"a route must at least have two coordinates and not more then 100",
		)
	}
	correlationID, err := getCorrelationID(ctx)
	if err != nil {
		return nil, missingOrInvalidCorrelationIDStatus
	}

	// setting response header for client side tracing
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationID.String()))

	// set correlatioId for logging
	logger := p.logger.With(
		zap.String("correlation_id", correlationID.String()),
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
	if errors.Is(err, poi.ErrInvalidSearchCoordinates) {
		return nil, status.Errorf(codes.InvalidArgument, "%s: %v", invalidGeoParamsMessage, err)
	}
	if err != nil {
		logger.Error("unable to handle request", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "%s: %v", severErrMessage, err)
	}

	// log and return
	logger.Info(
		"returning response for Route RPC",
		zap.Int("num_locations", len(locations)),
	)
	resp := buildPoISearchResponse(locations)
	return resp, nil
}

func (p *PoIRPCService) Register(server *grpc.Server) {
	poi_v1.RegisterPoIServiceServer(server, p)
}

func (p *PoIRPCService) RegisterProxy(
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

func buildPoISearchResponse(l []*poi.PoILocation) *poi_v1.PoISearchResponse {
	proto := locationsToProto(l)
	return &poi_v1.PoISearchResponse{
		Items: proto,
	}
}

func locationsToProto(l []*poi.PoILocation) []*poi_v1.PoI {
	pois := make([]*poi_v1.PoI, len(l))
	for i, v := range l {
		pois[i] = poiToProto(v)
	}
	return pois
}

func poiToProto(p *poi.PoILocation) *poi_v1.PoI {
	return &poi_v1.PoI{
		Id: p.ID.String(),
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
