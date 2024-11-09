package core

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/adapters/dynamo"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/adapters/rpc"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/app"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/domain/poi"
)

type ApplicationRunner struct {
	ctx        context.Context
	logger     *zap.Logger
	bootConfig *app.BootConfig
	server     *rpc.Server
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
				"unable to create application since boot config can not be loaded, please check boot.yaml location is as expected and permissions to read the file are given: %w",
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
	if strings.ToLower(a.bootConfig.Logging.Level) == "prod" {
		a.logger = app.NewLogger()
	} else {
		a.logger = app.NewDevLogger()
	}
	repo, err := a.createRepo()
	if err != nil {
		panic(fmt.Errorf("unable to create repo: %w", err))
	}
	domainService := poi.NewLocationService(repo, a.logger)
	serverOpts := a.getSevrerBaseOptions()
	serverOpts = append(
		serverOpts,
		rpc.WithHealthService(&rpc.HealthRpcService{}),
		rpc.WithRegisterRpcService(rpc.NewPoIRpcService(a.logger, domainService)),
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
	if a.bootConfig.Aws.DynamoDb.EndpointOverride.Enabled {
		dynamoOpts = append(dynamoOpts,
			dynamo.WithEndPointOverride(
				a.bootConfig.Aws.DynamoDb.EndpointOverride.Host,
				a.bootConfig.Aws.DynamoDb.EndpointOverride.Port,
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
		dynamo.WithDynamoClientWrapper(dyanmoClient),
		dynamo.WithTableName(a.bootConfig.Aws.DynamoDb.PoiTableName),
		dynamo.WithCreateAndInitTable(a.bootConfig.Aws.DynamoDb.CreateInitTable),
		dynamo.WithLogger(a.logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Repository: %w", err)
	}
	return repo, nil
}

func (a *ApplicationRunner) getSevrerBaseOptions() []rpc.ServerOption {
	return []rpc.ServerOption{
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
	}
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
	a.awaitTermination()
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
