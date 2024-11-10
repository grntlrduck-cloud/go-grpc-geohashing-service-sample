package rpc

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"

	poi_v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/poi"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/domain/poi"
)

var (
	missingCorrelationIdStatus = status.Errorf(
		codes.InvalidArgument,
		"invalid request, X-Correlation-Id header is required",
	)
	severErrStatus = status.Errorf(codes.Internal, "server error, failed to process request")
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
	if request == nil || request.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing argument id")
	}
	correlationId, err := getCorrelationId(ctx)
	if err != nil {
		return nil, missingCorrelationIdStatus
	}
	kId, err := ksuid.Parse(request.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid location id format")
	}

	p.logger.Info(
		"processing PoI rpc",
		zap.String("id", request.Id),
		zap.String(correlationHeader, correlationId.String()),
	)
	location, err := p.locationService.Info(ctx, kId, correlationId)
	if err != nil {
		return nil, severErrStatus
	}
	if location.Id == ksuid.Max {
		return nil, status.Errorf(codes.NotFound, "location not found")
	}

	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationId.String()))
	response := poi_v1.PoIResponse{
		Poi: poiToProto(location),
	}
	p.logger.Info(
		"returning response for PoI rpc",
		zap.String("id", request.Id),
		zap.String(correlationHeader, correlationId.String()),
	)
	return &response, nil
}

func (p *PoIRpcService) Proximity(
	ctx context.Context,
	request *poi_v1.ProximityRequest,
) (*poi_v1.ProximityResponse, error) {
	if request == nil || request.Center == nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"center must be given to perform proximity search",
		)
	}
	correlationId, err := getCorrelationId(ctx)
	if err != nil {
		return nil, missingCorrelationIdStatus
	}
	cntr := poi.Coordinates{
		Latitude:  request.Center.Lat,
		Longitude: request.Center.Lon,
	}
	p.logger.Info(
		"processing Proximity rpc",
		zap.String("correlation_id", correlationId.String()),
	)
	locations, err := p.locationService.Proximity(ctx, cntr, correlationId)
	if err != nil {
		return nil, severErrStatus
	}
	proto := locationsToProto(locations)
	p.logger.Info(
		"returning response for Proximity RPC",
		zap.Int("num_locations", len(proto)),
		zap.String("correlation_id", correlationId.String()),
	)
	resp := poi_v1.ProximityResponse{
		Items: proto,
	}
	return &resp, nil
}

func (p *PoIRpcService) BBox(
	ctx context.Context,
	request *poi_v1.BBoxRequest,
) (*poi_v1.BBoxResponse, error) {
	if request == nil || request.Bbox == nil || request.Bbox.Sw == nil || request.Bbox.Ne == nil {
		return nil, status.Errorf(codes.InvalidArgument, "bounding box coordinates must be defined")
	}
	correlationId, err := getCorrelationId(ctx)
	if err != nil {
		return nil, missingCorrelationIdStatus
	}
	sw := poi.Coordinates{
		Latitude:  request.Bbox.Sw.Lat,
		Longitude: request.Bbox.Sw.Lon,
	}
	ne := poi.Coordinates{
		Latitude:  request.Bbox.Ne.Lat,
		Longitude: request.Bbox.Ne.Lon,
	}
	p.logger.Info(
		"processing BBox rpc",
		zap.String("correlation_id", correlationId.String()),
	)
	locations, err := p.locationService.Bbox(ctx, sw, ne, correlationId)
	if err != nil {
		return nil, severErrStatus
	}
	proto := locationsToProto(locations)
	p.logger.Info(
		"returning response for BBox RPC",
		zap.Int("num_locations", len(proto)),
		zap.String("correlation_id", correlationId.String()),
	)
	resp := poi_v1.BBoxResponse{
		Items: proto,
	}
	return &resp, nil
}

func (p *PoIRpcService) Route(
	ctx context.Context,
	request *poi_v1.RouteRequest,
) (*poi_v1.RouteResponse, error) {
	if request == nil || request.Route == nil || len(request.Route) < 2 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"a route og at least two coordinates must be provided",
		)
	}
	correlationId, err := getCorrelationId(ctx)
	if err != nil {
		return nil, missingCorrelationIdStatus
	}
	path := coordinatesPathFromProto(request.Route)
	p.logger.Info(
		"processing Route rpc",
		zap.String("correlation_id", correlationId.String()),
	)
	locations, err := p.locationService.Route(ctx, path, correlationId)
	if err != nil {
		return nil, severErrStatus
	}
	proto := locationsToProto(locations)
	p.logger.Info(
		"returning response for Route RPC",
		zap.Int("num_locations", len(proto)),
		zap.String("correlation_id", correlationId.String()),
	)
	resp := poi_v1.RouteResponse{
		Items: proto,
	}
	return &resp, nil
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
