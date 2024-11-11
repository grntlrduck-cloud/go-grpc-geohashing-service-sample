package rpc

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Service interface {
	Register(server *grpc.Server)
	RegisterProxy(
		ctx context.Context,
		mux *runtime.ServeMux,
		endpoint string,
		opts []grpc.DialOption,
	) (err error)
}

type HealthChecker interface {
	SetHealth(healthy bool)
}

type HealthService interface {
  Service
  HealthChecker
}
