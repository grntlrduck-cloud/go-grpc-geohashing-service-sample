package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/rpc"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}
	logger.Info("initalized logger")
}

func main() {
	defer func() {
		_ = logger.Sync()
	}()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	serverConfig := rpc.ServerConfig{RpcPort: 9091, HttpPort: 8081}
	server, err := rpc.StartNewServer(ctx, serverConfig, logger)
	if err != nil {
		logger.Panic("failed to start rRPC server and reverse proxy for HTTP/json")
	}
	defer server.Stop()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	logger.Info("running and serving requests")
	for {
		select {
		case <-c:
			return
		case <-ctx.Done():
			return
		default:
			time.Sleep(100 * time.Microsecond)
		}
	}
}
