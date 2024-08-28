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
	Conf   app.ServerConfig
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

type ServerStartFailureErr struct {
  e error
}

func (se ServerStartFailureErr) Error() string {
  return fmt.Sprintf("fatal, failed to start gRPC gateway server: %s", se.e)
 }

func (s *Server) Stop() {
	s.hrs.healthy(false)
	s.logger.Info("set health endpoint to NOT_SERVING")
	time.Sleep(1 * time.Second)
	s.rpcServer.GracefulStop()
	s.logger.Info("stopped gRPC server gracefully")
}

func NewServer(props NewServerProps) (*Server, error) {
	grpcServerEndpoint := fmt.Sprintf("localhost:%d", props.Conf.RpcPort)
	httpProxyEndpoint := fmt.Sprintf("localhost:%d", props.Conf.HttpPort)
	
  // start rpc server and add service
	res := startRpcServer(props.Logger, grpcServerEndpoint)
	if res.err != nil {
		props.Logger.Error("failed to start gRPC server")
		return nil, ServerStartFailureErr{res.err}
	}

	// start http proxy
	err := startHttpProxy(startHttpProxyProps{
		logger:       props.Logger,
		ctx:          props.Ctx,
		rpcEndpoint:  grpcServerEndpoint,
		httpEndpoint: httpProxyEndpoint,
	})
	if err != nil {
		props.Logger.Error("failed to start http proxy, stopping gRPC server", zap.Error(err))
		res.rpcServer.GracefulStop()
		return nil, ServerStartFailureErr{err}
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
		logger.Error("failed to listen to port", zap.Error(err))
		return &startRpcServerResult{nil, nil, nil, err}
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
			logger.Panic("failed to start rpc server")
		}
	}()
	return &startRpcServerResult{server, hrs, prs, nil}
}

// https://github.com/grpc-ecosystem/go-grpc-middleware/blob/main/interceptors/logging/examples/zap/example_test.go
func InterceptorLogger(l *zap.Logger) grpclogging.Logger {
	return grpclogging.LoggerFunc(
		func(ctx context.Context, lvl grpclogging.Level, msg string, fields ...any) {
			f := make([]zap.Field, 0, len(fields)/2)

			for i := 0; i < len(fields); i += 2 {
				key := fields[i]
				value := fields[i+1]

				switch v := value.(type) {
				case string:
					f = append(f, zap.String(key.(string), v))
				case int:
					f = append(f, zap.Int(key.(string), v))
				case bool:
					f = append(f, zap.Bool(key.(string), v))
				default:
					f = append(f, zap.Any(key.(string), v))
				}
			}

			logger := l.WithOptions(zap.AddCallerSkip(1)).With(f...)

			switch lvl {
			case grpclogging.LevelDebug:
				logger.Debug(msg)
			case grpclogging.LevelInfo:
				logger.Debug(msg)
			case grpclogging.LevelWarn:
				logger.Warn(msg)
			case grpclogging.LevelError:
				logger.Error(msg)
			default:
				panic(fmt.Sprintf("unknown level %v", lvl))
			}
		},
	)
}

func startHttpProxy(props startHttpProxyProps) error {
	props.logger.Info("starting HTTP reverse proxy with RPC handler")
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(CustomMatcher),
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
		props.logger.Error(
			"failed to register http revers proxy handler for poi service",
			zap.Error(err),
		)
		return err
	}
	err = health_v1.RegisterHealthServiceHandlerFromEndpoint(
		props.ctx,
		mux,
		props.rpcEndpoint,
		opts,
	)
	if err != nil {
		props.logger.Error(
			"failed to register http revers proxy handler for health service",
			zap.Error(err),
		)
		return err
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

func CustomMatcher(key string) (string, bool) {
	switch key {
	case "X-Correlation-Id":
		return key, true
	default:
		return key, false
	}
}
