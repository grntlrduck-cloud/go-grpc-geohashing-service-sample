package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/app"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/core"
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

	rnnr, err := core.NewApplicationRunner(
		core.WithApplicationContext(ctx),
		core.WithApplicationLogger(logger),
	)
	if err != nil {
		logger.Panic("failed start failure", zap.Error(err))
	}
	rnnr.Run()
	logger.Info("shut down")
}
