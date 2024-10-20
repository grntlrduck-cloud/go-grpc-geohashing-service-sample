package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	grpclogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	health_v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/health"
	poi_v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/poi"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/app"
)

type Server struct {
	prs       *PoIRpcService
	hrs       *HealthRpcService
	rpcServer *grpc.Server
	logger    *zap.Logger
}

type NewServerProps struct {
	Logger *zap.Logger
	Ctx    context.Context
	Conf   app.GrpcConfig
}

type startHttpProxyProps struct {
	logger       *zap.Logger
	ctx          context.Context
	rpcEndpoint  string
	httpEndpoint string
}

type startRpcServerResult struct {
	rpcServer *grpc.Server
	hrs       *HealthRpcService
	prs       *PoIRpcService
	err       error
}

func (s *Server) Stop() {
	s.hrs.healthy(false)
	s.logger.Info("set health endpoint to NOT_SERVING")

	// sleep to stop ALB from sending requests to this instance
	time.Sleep(1 * time.Second)

	s.rpcServer.GracefulStop()
	s.logger.Info("stopped gRPC server gracefully")
}

func NewServer(props NewServerProps) (*Server, error) {
	grpcServerEndpoint := fmt.Sprintf(":%d", props.Conf.Server.Port)
	httpProxyEndpoint := fmt.Sprintf(":%d", props.Conf.Proxy.Port)
	// start rpc server and add service
	res := startRpcServer(props.Logger, grpcServerEndpoint)
	if res.err != nil {
		return nil, fmt.Errorf("failed to start rpc server: %w", res.err)
	}

	// start http proxy
	err := startHttpProxy(startHttpProxyProps{
		logger:       props.Logger,
		ctx:          props.Ctx,
		rpcEndpoint:  grpcServerEndpoint,
		httpEndpoint: httpProxyEndpoint,
	})
	if err != nil {
		res.rpcServer.GracefulStop()
		return nil, fmt.Errorf("failed to start reverse rpc proxy: %w", err)
	}
	props.Logger.Info("setting endpoints to SERVING")
	res.hrs.healthy(true)

	props.Logger.Info("finished initializing server")
	return &Server{hrs: res.hrs, prs: res.prs, rpcServer: res.rpcServer, logger: props.Logger}, nil
}

func startRpcServer(logger *zap.Logger, endpoint string) *startRpcServerResult {
	logger.Info("starting gRPC server")
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return &startRpcServerResult{
			nil,
			nil,
			nil,
			fmt.Errorf("failed to listern tcp port %s: %w", endpoint, err),
		}
	}
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpclogging.UnaryServerInterceptor(InterceptorLogger(logger)),
		),
		grpc.ChainStreamInterceptor(
			grpclogging.StreamServerInterceptor(InterceptorLogger(logger)),
		),
	)
	hrs := &HealthRpcService{}
	health_v1.RegisterHealthServiceServer(server, hrs)
	hrs.healthy(false)
	prs := &PoIRpcService{logger: logger}
	poi_v1.RegisterPoIServiceServer(server, prs)
	go func() {
		servErr := server.Serve(lis)
		if servErr != nil {
			logger.Panic("failed to start rpc server", zap.Error(servErr))
		}
	}()
	return &startRpcServerResult{server, hrs, prs, nil}
}

func startHttpProxy(props startHttpProxyProps) error {
	props.logger.Info("starting HTTP reverse proxy with RPC handler")
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(correlationIdMatcher),
		runtime.WithForwardResponseOption(correlationIdResponseModifier),
	)
	opts := []grpc.DialOption{
		// TODO: configure security based on props
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// use zap logger for handlers
		grpc.WithChainUnaryInterceptor(
			grpclogging.UnaryClientInterceptor(InterceptorLogger(props.logger)),
		),
		grpc.WithChainStreamInterceptor(
			grpclogging.StreamClientInterceptor(InterceptorLogger(props.logger)),
		),
	}
	err := poi_v1.RegisterPoIServiceHandlerFromEndpoint(props.ctx, mux, props.rpcEndpoint, opts)
	if err != nil {
		return fmt.Errorf("failed to register poi handler: %w", err)
	}
	err = health_v1.RegisterHealthServiceHandlerFromEndpoint(
		props.ctx,
		mux,
		props.rpcEndpoint,
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to register health handler: %w", err)
	}
	// Start HTTP server (and proxy calls to gRPC server endpoint)
	go func() {
		err = http.ListenAndServe(props.httpEndpoint, mux)
		if err != nil {
			props.logger.Panic("failed to start http server", zap.Error(err))
		}
	}()
	return nil
}
