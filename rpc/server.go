package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1"
)

type Server struct {
	prs       *PoIRpcService
	rpcServer *grpc.Server
	logger    *zap.Logger
}

// To be moved to config package later on
type ServerConfig struct {
	RpcPort  int32
	HttpPort int32
}

func (s *Server) Stop() {
	s.rpcServer.GracefulStop()
	s.logger.Info("stopped gRPC server gracefully")
}

func StartNewServer(ctx context.Context, config ServerConfig, logger *zap.Logger) (*Server, error) {
	// Create a listener on TCP port
	grpcServerEndpoint := fmt.Sprintf("localhost:%d", config.RpcPort)
	lis, err := net.Listen("tcp", grpcServerEndpoint)
	if err != nil {
		logger.Error("failed to listen to port")
		return nil, err
	}

	// TODO: custom implementation for error handler
  // TODO: custom implementation for response hesders to map correlationId
	// start rpc server and attach PoIRpcService
	logger.Info("starting gRPC server")
	server := grpc.NewServer()
	prs := &PoIRpcService{logger: logger}
	v1.RegisterPoIServiceServer(server, prs)
	go func(s *grpc.Server) {
		servErr := server.Serve(lis)
		if servErr != nil {
			logger.Panic("failed to start rpc server")
		}
	}(server)

	// register the http reverse proxy handler and start the http server
	logger.Info("starting HTTP reverse proxy with RPC handler")
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = v1.RegisterPoIServiceHandlerFromEndpoint(ctx, mux, grpcServerEndpoint, opts)
	if err != nil {
		logger.Error("failed to register http revers proxy handler")
		return nil, err
	}
	// Start HTTP server (and proxy calls to gRPC server endpoint)
	httpServerEndpoint := fmt.Sprintf(":%d", config.HttpPort)
  go func () {
    err = http.ListenAndServe(httpServerEndpoint, mux)
	  if err != nil {
		 logger.Panic("failed to start http server")
	  }
  }()
	logger.Info("finished initializing server")
	return &Server{prs: prs, rpcServer: server, logger: logger}, nil
}
