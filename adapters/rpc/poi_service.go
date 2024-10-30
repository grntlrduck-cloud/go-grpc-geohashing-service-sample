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
	correlationId, err := getCorrelationId(ctx)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid request, X-Correlation-Id header is required",
		)
	}
	kId, err := ksuid.Parse(request.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid location id")
	}

	p.logger.Info(
		"processing PoI rpc",
		zap.String("id", request.Id),
		zap.String(correlationHeader, correlationId.String()),
	)
	location, err := p.locationService.Info(ctx, kId, correlationId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "server error")
	}
	if location.Id == ksuid.Max {
		return nil, status.Errorf(codes.NotFound, "location not found")
	}
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, correlationId.String()))
	response := poi_v1.PoIResponse{
		Poi: domainToProto(location),
	}
	return &response, nil
}

func (prs *PoIRpcService) Proximity(
	ctex context.Context,
	request *poi_v1.ProximityRequest,
) (*poi_v1.ProximityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Proximity not implemented")
}

func (p *PoIRpcService) BBox(
	ctx context.Context,
	request *poi_v1.BBoxRequest,
) (*poi_v1.BBoxResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BBox not implemented")
}

func (p *PoIRpcService) Route(
	ctx context.Context,
	request *poi_v1.RouteRequest,
) (*poi_v1.RouteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Route not implemented")
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

func domainToProto(location poi.PoILocation) *poi_v1.PoI {
	return &poi_v1.PoI{
		Id: location.Id.String(),
		Coordinate: &poi_v1.Coordinate{
			Lat: location.Location.Latitude,
			Lon: location.Location.Longitude,
		},
		Address: &poi_v1.Address{
			Street:       location.Address.Street,
			StreetNumber: location.Address.StreetNumber,
			ZipCode:      location.Address.ZipCode,
			City:         location.Address.City,
			Country:      location.Address.CountryCode,
		},
		Features: location.Features,
	}
}
