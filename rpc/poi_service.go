package rpc

import (
	"context"

	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1"
)

type PoIRpcService struct {
	v1.UnimplementedPoIServiceServer
	logger *zap.Logger
}

func (prs *PoIRpcService) PoI(
	ctx context.Context,
	request *v1.PoIRequest,
) (*v1.PoIResponse, error) {
	prs.logger.Info("processing PoI rpc, returning fixture")
	return &v1.PoIResponse{
		Poi: &v1.PoI{
			Id:         ksuid.New().String(),
			Coordinate: &v1.Coordinate{Lat: 48.137, Lon: 11.576},
			Address: &v1.Address{
				Street:       "Maximilianstrasse",
				StreetNumber: "1b",
				ZipCode:      "123456",
				City:         "Munich",
				Country:      "DEU",
			},
		},
	}, nil
}

func (prs *PoIRpcService) Proximity(
	ctex context.Context,
	request *v1.ProximityRequest,
) (*v1.ProximityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Proximity not implemented")
}

func (prs *PoIRpcService) BBox(
	ctx context.Context,
	request *v1.BBoxRequest,
) (*v1.BBoxResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BBox not implemented")
}

func (prs *PoIRpcService) Route(
	ctx context.Context,
	request *v1.RouteRequest,
) (*v1.RouteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Route not implemented")
}
