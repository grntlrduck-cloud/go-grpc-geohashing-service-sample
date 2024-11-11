package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/core"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rnnr := core.NewApplicationRunner(
		core.WithApplicationContext(ctx),
	)

	rnnr.Run()
}
