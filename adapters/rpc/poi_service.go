package rpc

import (
	"context"

	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"

	poi_v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/poi"
)

type PoIRpcService struct {
	poi_v1.UnimplementedPoIServiceServer
	logger *zap.Logger
}

func (prs *PoIRpcService) PoI(
	ctx context.Context,
	request *poi_v1.PoIRequest,
) (*poi_v1.PoIResponse, error) {
	id, err := getCorrelationId(ctx)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"correlationId is required",
		)
	}
	prs.logger.Info(
		"processing PoI rpc, returning fixture",
		zap.String(correlationHeader, id.String()),
	)
	_ = grpc.SendHeader(ctx, metadata.Pairs(correlationHeader, id.String()))
	return &poi_v1.PoIResponse{
		Poi: &poi_v1.PoI{
			Id:         ksuid.New().String(),
			Coordinate: &poi_v1.Coordinate{Lat: 48.137, Lon: 11.576},
			Address: &poi_v1.Address{
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
	request *poi_v1.ProximityRequest,
) (*poi_v1.ProximityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Proximity not implemented")
}

func (prs *PoIRpcService) BBox(
	ctx context.Context,
	request *poi_v1.BBoxRequest,
) (*poi_v1.BBoxResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BBox not implemented")
}

func (prs *PoIRpcService) Route(
	ctx context.Context,
	request *poi_v1.RouteRequest,
) (*poi_v1.RouteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Route not implemented")
}
