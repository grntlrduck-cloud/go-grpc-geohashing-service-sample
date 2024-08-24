package rpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	id, err := prs.getCorrelationId(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Header X-Correlation-Id not found")
	}
	prs.logger.Info(
		"processing PoI rpc, returning fixture",
		zap.String("X-Correlation-Id", id.String()),
	)
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

func (prs *PoIRpcService) getCorrelationId(ctx context.Context) (*uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		prs.logger.Error("failed to get metadata from ctx")
		return nil, errors.New("failed to extract metadata in service")
	}
	match := md.Get("X-Correlation-Id")
	if len(match) > 0 {
		id, err := uuid.Parse(match[0])
		if err != nil {
			prs.logger.Warn("failed to parse uuid form correlation header")
			return nil, err
		}
		return &id, nil
	}
	prs.logger.Warn("no correlationId set")
	return nil, errors.New("correlationId not in headers")
}
