package main

import (
	"context"
	"fmt"
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

	application, err := NewApplication(
		WithApplicationContext(ctx),
		WithApplicationLogger(logger),
	)
	if err != nil {
		logger.Panic("failed start failure", zap.Error(err))
	}
	application.Run()
	logger.Info("shut down")
}

type Application struct {
	ctx        context.Context
	logger     *zap.Logger
	bootConfig *app.BootConfig
	server     *rpc.Server
}

type ApplicationOpts func(*Application)

func WithApplicationContext(ctx context.Context) ApplicationOpts {
	return func(a *Application) {
		a.ctx = ctx
	}
}

func WithApplicationLogger(logger *zap.Logger) ApplicationOpts {
	return func(a *Application) {
		a.logger = logger
	}
}

func WithBootConfigOverride(conf *app.BootConfig) ApplicationOpts {
	return func(a *Application) {
		a.bootConfig = conf
	}
}

func NewApplication(opts ...ApplicationOpts) (*Application, error) {
	conf, err := app.LoadBootConfig()
	if err != nil {
		return nil, fmt.Errorf(
			"unable to create application since boot config can not be loaded, please check boot.yaml location is as expected and permissions to read the file are given: %w",
			err,
		)
	}
	a := &Application{
		ctx:        context.Background(),
		bootConfig: conf,
	}
	for _, opt := range opts {
		opt(a)
	}
	if a.logger == nil {
		a.logger = app.NewDevLogger()
	}
	server, err := rpc.NewServer(
		rpc.WithContext(a.ctx),
		rpc.WithRpcLogger(a.logger),
		rpc.WithGrpcPort(a.bootConfig.Grpc.Server.Port),
		rpc.WithHttpPort(a.bootConfig.Grpc.Proxy.Port),
		rpc.WithSslConfig(
			a.bootConfig.Grpc.Ssl.CertPath,
			a.bootConfig.Grpc.Ssl.KeyPath,
			a.bootConfig.Grpc.Ssl.CaPath,
		),
		rpc.WithSslEnabled(a.bootConfig.Grpc.Ssl.Enabled),
		rpc.WithHealthService(&rpc.HealthRpcService{}),
		rpc.WithRegisterRpcService(rpc.NewPoIRpcService(a.logger)),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create application, unable to create server from boot config: %w",
			err,
		)
	}
	a.server = server
	return a, nil
}

func (a *Application) Run() {
	defer a.server.Stop()
	err := a.server.Start()
	if err != nil {
		a.logger.Panic("application run failed, unable to start grpc server", zap.Error(err))
	}
	a.logger.Info("application running")
	a.awaitTermination()
}

func (a *Application) awaitTermination() {
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			continue
		}
	}
}
