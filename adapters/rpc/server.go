package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"

	grpclogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1"
  "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/app"
)

type Server struct {
	prs       *PoIRpcService
	rpcServer *grpc.Server
	logger    *zap.Logger
}

type NewServerProps struct {
  Logger *zap.Logger
  Ctx context.Context
  Conf app.ServerConfig
}

type startHttpProxyProps struct {
	logger       *zap.Logger
	ctx          context.Context
	rpcEndpoint  string
	httpEndpoint string
}

type startRpcServerResult struct {
	rpcServer *grpc.Server
	prs       *PoIRpcService
	err       error
}

func (s *Server) Stop() {
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
		return nil, res.err
	}

	// start http proxy
	err := startHttProxy(startHttpProxyProps{
		logger:       props.Logger,
		ctx:          props.Ctx,
		rpcEndpoint:  grpcServerEndpoint,
		httpEndpoint: httpProxyEndpoint,
	})
	if err != nil {
		props.Logger.Error("failed to start http proxy, stopping gRPC server")
		res.rpcServer.GracefulStop()
		return nil, err
	}

	props.Logger.Info("finished initializing server")
	return &Server{prs: res.prs, rpcServer: res.rpcServer, logger: props.Logger}, nil
}

func startRpcServer(logger *zap.Logger, endpoint string) *startRpcServerResult {
	logger.Info("starting gRPC server")
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		logger.Error("failed to listen to port")
		return &startRpcServerResult{nil, nil, err}
	}
	grpcOpts := []grpclogging.Option{
		grpclogging.WithLogOnEvents(grpclogging.StartCall, grpclogging.FinishCall),
	}
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpclogging.UnaryServerInterceptor(InterceptorLogger(logger), grpcOpts...),
		),
		grpc.ChainStreamInterceptor(
			grpclogging.StreamServerInterceptor(InterceptorLogger(logger), grpcOpts...),
		),
	)
	prs := &PoIRpcService{logger: logger}
	v1.RegisterPoIServiceServer(server, prs)
	go func() {
		servErr := server.Serve(lis)
		if servErr != nil {
			logger.Panic("failed to start rpc server")
		}
	}()
	return &startRpcServerResult{server, prs, nil}
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
				logger.Info(msg)
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

func startHttProxy(props startHttpProxyProps) error {
	props.logger.Info("starting HTTP reverse proxy with RPC handler")
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(CustomMatcher),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := v1.RegisterPoIServiceHandlerFromEndpoint(props.ctx, mux, props.rpcEndpoint, opts)
	if err != nil {
		props.logger.Error("failed to register http revers proxy handler")
		return err
	}
	// Start HTTP server (and proxy calls to gRPC server endpoint)
	go func() {
		err = http.ListenAndServe(props.httpEndpoint, mux)
		if err != nil {
			props.logger.Panic("failed to start http server")
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
