package rpc

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	health_v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/health"
)

type HealthRPCService struct {
	health_v1.UnimplementedHealthServiceServer
	serverHealthy bool
}

func (h *HealthRPCService) HealthCheck(
	ctx context.Context,
	request *health_v1.HealthCheckRequest,
) (*health_v1.HealthCheckResponse, error) {
	if h.serverHealthy {
		return &health_v1.HealthCheckResponse{
			Status: health_v1.HealthCheckResponse_SERVING_STATUS_SERVING,
		}, nil
	}
	return nil, status.Errorf(codes.Unavailable, "endpoints not available")
}

func (h *HealthRPCService) SetHealth(healthy bool) {
	h.serverHealthy = healthy
}

func (h *HealthRPCService) Register(server *grpc.Server) {
	health_v1.RegisterHealthServiceServer(server, h)
}

func (h *HealthRPCService) RegisterProxy(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return health_v1.RegisterHealthServiceHandlerFromEndpoint(
		ctx,
		mux,
		endpoint,
		opts,
	)
}
