package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/adapters/rpc"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/app"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}
	logger.Info("initialized logger")
}

func main() {
	defer func() {
		_ = logger.Sync()
	}()

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	serverConfig := app.ServerConfig{RpcPort: 9091, HttpPort: 8081}
	server, err := rpc.NewServer(rpc.NewServerProps{
		Logger: logger,
		Ctx:    ctx,
		Conf:   serverConfig,
	})
	if err != nil {
		logger.Panic("failed to start rRPC server and reverse proxy for HTTP/json")
	}
	defer server.Stop()

	logger.Info("running and serving requests")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(100 * time.Microsecond)
		}
	}
}
