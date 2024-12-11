package core

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/adapters/dynamo"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/adapters/rpc"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/app"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/domain/poi"
)

type ApplicationRunner struct {
	ctx        context.Context
	logger     *zap.Logger
	bootConfig *app.BootConfig
	server     *rpc.Server
	running    bool
}

type ApplicationOpts func(*ApplicationRunner)

func WithApplicationContext(ctx context.Context) ApplicationOpts {
	return func(a *ApplicationRunner) {
		a.ctx = ctx
	}
}

func WithBootConfigOverride(conf *app.BootConfig) ApplicationOpts {
	return func(a *ApplicationRunner) {
		a.bootConfig = conf
	}
}

func NewApplicationRunner(opts ...ApplicationOpts) *ApplicationRunner {
	conf, err := app.LoadBootConfig()
	if err != nil {
		panic(
			fmt.Errorf(
				"unable to create application since boot config: %w",
				err,
			),
		)
	}
	a := &ApplicationRunner{
		ctx:        context.Background(),
		bootConfig: conf,
	}
	for _, opt := range opts {
		opt(a)
	}
	if strings.EqualFold(a.bootConfig.Logging.Level, "prod") {
		a.logger = app.NewLogger(&a.bootConfig.Logging)
	} else {
		a.logger = app.NewDevLogger(&a.bootConfig.Logging)
	}
	repo, err := a.createRepo()
	if err != nil {
		panic(fmt.Errorf("unable to create repo: %w", err))
	}
	domainService := poi.NewLocationService(repo)
	serverOpts := a.getSevrerBaseOptions()
	serverOpts = append(
		serverOpts,
		rpc.WithHealthService(&rpc.HealthRPCService{}),
		rpc.WithRegisterRPCService(rpc.NewPoIRPCService(a.logger, domainService)),
	)
	server, err := rpc.NewServer(serverOpts...)
	if err != nil {
		panic(fmt.Errorf(
			"failed to create application, unable to create server from boot config: %w",
			err,
		),
		)
	}
	a.server = server
	return a
}

func (a *ApplicationRunner) createRepo() (poi.Repository, error) {
	dynamoOpts := []dynamo.ClientOptions{
		dynamo.WithContext(a.ctx),
		dynamo.WithRegion(a.bootConfig.Aws.Config.Region),
	}
	if a.bootConfig.Aws.DynamoDB.EndpointOverride.Enabled {
		dynamoOpts = append(dynamoOpts,
			dynamo.WithEndPointOverride(
				a.bootConfig.Aws.DynamoDB.EndpointOverride.Host,
				a.bootConfig.Aws.DynamoDB.EndpointOverride.Port,
			),
		)
	}
	dyanmoClient, err := dynamo.NewClientWrapper(dynamoOpts...)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create dynamoDb client, is IAM confiured correctlty? %w",
			err,
		)
	}
	repo, err := dynamo.NewPoIGeoRepository(
		a.logger,
		dynamo.WithDynamoClientWrapper(dyanmoClient),
		dynamo.WithTableName(a.bootConfig.Aws.DynamoDB.PoiTableName),
		dynamo.WithCreateAndInitTable(a.bootConfig.Aws.DynamoDB.CreateInitTable),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Repository: %w", err)
	}
	return repo, nil
}

func (a *ApplicationRunner) getSevrerBaseOptions() []rpc.ServerOption {
	return []rpc.ServerOption{
		rpc.WithContext(a.ctx),
		rpc.WithRPCLogger(a.logger),
		rpc.WithGRPCPort(a.bootConfig.Grpc.Server.Port),
		rpc.WithHTTPPort(a.bootConfig.Grpc.Proxy.Port),
		rpc.WithSSLConfig(
			a.bootConfig.Grpc.Ssl.CertPath,
			a.bootConfig.Grpc.Ssl.KeyPath,
			a.bootConfig.Grpc.Ssl.CaPath,
		),
		rpc.WithSSLEnabled(a.bootConfig.Grpc.Ssl.Enabled),
		rpc.WithAuthSecret(a.bootConfig.Grpc.Secret),
	}
}

func (a *ApplicationRunner) Running() bool {
	return a.running
}

func (a *ApplicationRunner) BootConfig() *app.BootConfig {
	return a.bootConfig
}

func (a *ApplicationRunner) Run() {
	defer func(a *ApplicationRunner) {
		_ = a.logger.Sync()
	}(a)
	defer a.server.Stop()
	err := a.server.Start()
	if err != nil {
		a.logger.Panic("application run failed, unable to start grpc server", zap.Error(err))
	}
	a.logger.Info("application running")
	a.running = true
	a.awaitTermination()
	a.running = false
	a.logger.Info("application shut down")
}

func (a *ApplicationRunner) awaitTermination() {
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			continue
		}
	}
}
