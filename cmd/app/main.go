package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/adapters/rpc"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/app"
)

var logger *zap.Logger

func init() {
	logger = app.NewLogger()
}

func main() {
	defer func() {
		_ = logger.Sync()
	}()

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	bootConfig, err := app.LoadBootConfig()
	if err != nil {
		logger.Panic("failed to start application. unable to load boot config", zap.Error(err))
	}

	server, err := rpc.NewServer(
		rpc.WithContext(ctx),
		rpc.WithRpcLogger(logger),
		rpc.WithGrpcPort(bootConfig.Grpc.Server.Port),
		rpc.WithHttpPort(bootConfig.Grpc.Proxy.Port),
		rpc.WithSslConfig(
			bootConfig.Grpc.Ssl.CertPath,
			bootConfig.Grpc.Ssl.KeyPath,
			bootConfig.Grpc.Ssl.CaPath,
		),
		rpc.WithSslEnabled(bootConfig.Grpc.Ssl.Enabled),
		rpc.WithHealthService(&rpc.HealthRpcService{}),
		rpc.WithRegisterRpcService(rpc.NewPoIRpcService(logger)),
	)
	if err != nil {
		logger.Panic("failed to creat gRPC server", zap.Error(err))
	}
	defer server.Stop()
	err = server.Start()
	if err != nil {
		logger.Panic("failed to start gRPC Server and proxy gateway", zap.Error(err))
	}
	logger.Info("running and serving requests")

	awaitTermination(ctx)

	logger.Info("shutting down")
}

func awaitTermination(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}
