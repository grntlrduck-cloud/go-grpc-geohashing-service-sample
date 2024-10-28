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

	server, err := rpc.NewServer(rpc.NewServerProps{
		Logger: logger,
		Ctx:    ctx,
		Conf:   bootConfig.Grpc,
	})
	if err != nil {
		logger.Panic("failed to start rRPC server and reverse proxy for HTTP/JSON", zap.Error(err))
	}
	defer server.Stop()

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
