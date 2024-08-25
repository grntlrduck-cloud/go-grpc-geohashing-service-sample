package rpc

import (
	"context"

	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"

	health_v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/health"
)

type HealthRpcService struct {
	health_v1.UnimplementedHealthServiceServer
	serverHealthy bool
}

func (hrs *HealthRpcService) HealthCheck(
	ctx context.Context,
	request *health_v1.HealthCheckRequest,
) (*health_v1.HealthCheckResponse, error) {
	if hrs.serverHealthy {
		return &health_v1.HealthCheckResponse{
			Status: health_v1.HealthCheckResponse_SERVING_STATUS_SERVING,
		}, nil
	}
	return nil, status.Errorf(codes.Unavailable, "endpoints not available")
}

func (hrs *HealthRpcService) healthy(healthy bool) {
	hrs.serverHealthy = healthy
}
