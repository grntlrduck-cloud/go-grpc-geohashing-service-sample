package rpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	grpclogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/app"
)

const (
	defaultGrpcPort = 8081
	defaultHttpPort = 9081
)

type Server struct {
	healthService            HealthService
	rpcServer                *grpc.Server
	logger                   *zap.Logger
	services                 []Service
	ctx                      context.Context
	grpcPort                 int32
	httpPort                 int32
	sslEnabled               bool
	sslCertPath              string
	sslKeyPath               string
	sslCaPath                string
	grpcTlsConfig            credentials.TransportCredentials
	httpProxyTlsConfig       credentials.TransportCredentials
	albDeregistrationSeconds int64
}

type ServerOption func(s *Server)

func WithGrpcPort(port int32) ServerOption {
	return func(s *Server) {
		s.grpcPort = port
	}
}

func WithHttpPort(port int32) ServerOption {
	return func(s *Server) {
		s.httpPort = port
	}
}

func WithContext(ctx context.Context) ServerOption {
	return func(s *Server) {
		s.ctx = ctx
	}
}

func WithSslEnabled(sslEnabled bool) ServerOption {
	return func(s *Server) {
		s.sslEnabled = sslEnabled
	}
}

func WithRpcLogger(logger *zap.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

func WithSslConfig(certPath, keyPath, caPath string) ServerOption {
	return func(s *Server) {
		s.sslCertPath = certPath
		s.sslKeyPath = keyPath
		s.sslCaPath = caPath
	}
}

func WithRegisterRpcService(service Service) ServerOption {
	return func(s *Server) {
		if service != nil {
			s.services = append(s.services, service)
		}
	}
}

func WithHealthService(healthService HealthService) ServerOption {
	return func(s *Server) {
		if healthService != nil {
			s.healthService = healthService
		}
	}
}

func WithAlbDegistrationDelay(seconds int64) ServerOption {
	return func(s *Server) {
		if seconds > 0 {
			s.albDeregistrationSeconds = seconds
		}
	}
}

func NewServer(opts ...ServerOption) (*Server, error) {
	// apply defaults to server
	server := &Server{
		logger:                   app.NewDevLogger(),
		grpcPort:                 defaultGrpcPort,
		httpPort:                 defaultHttpPort,
		ctx:                      context.Background(),
		albDeregistrationSeconds: 1,
		services:                 make([]Service, 0),
	}
	// apply the options to the server with given overrides and configurations
	for _, opt := range opts {
		opt(server)
	}
	if server.healthService == nil {
		return nil, errors.New(
			"fatal, no health service registered. Use WithHealthService option to register health rpc",
		)
	}
	if server.sslEnabled && (server.sslKeyPath == "" || server.sslCertPath == "") {
		return nil, errors.New(
			"fatal, ssl is enabled but cert and key are not configured. Use WithSslCertConfig option to configure ssl",
		)
	}
	if server.sslEnabled {
		err := server.loadTlsConfig()
		if err != nil {
			return nil, fmt.Errorf("fatal, failed to load tls config: %w", err)
		}
	}
	if len(server.services) == 0 {
		server.logger.Warn(
			"no services registered for server. Use WithRegisterRpcService option to register services",
		)
	}
	return server, nil
}

func (s *Server) loadTlsConfig() error {
	s.logger.Info("loading tls config for server")
	ca, err := os.ReadFile(s.sslCaPath)
	if err != nil {
		return fmt.Errorf("umnable to load ca, given path might be incorrect: %w", err)
	}
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(ca) {
		return fmt.Errorf("failed to add server CA's certificate to pool of proxy client")
	}

	cert, err := tls.LoadX509KeyPair(s.sslCertPath, s.sslKeyPath)
	if err != nil {
		return err
	}

	s.grpcTlsConfig = credentials.NewTLS(
		&tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.NoClientCert,
			MinVersion:   tls.VersionTLS12,
		},
	)
	s.httpProxyTlsConfig = credentials.NewTLS(
		&tls.Config{
			RootCAs:    cp,
			MinVersion: tls.VersionTLS12,
		},
	)
	return nil
}

func (s *Server) Start() error {
	// explicit ensure we start the service unhealthy
	s.healthService.SetHealth(false)
	err := s.startRpcServer()
	if err != nil {
		return fmt.Errorf("failed to start rpc server: %w", err)
	}

	err = s.startHttpProxy()
	if err != nil {
		s.rpcServer.GracefulStop()
		return fmt.Errorf("failed to start reverse rpc proxy: %w", err)
	}

	s.logger.Info("setting endpoints to SERVING")
	s.healthService.SetHealth(true)

	s.logger.Info("finished initializing server")
	return nil
}

func (s *Server) startRpcServer() error {
	endpoint := fmt.Sprintf(":%d", s.grpcPort)
	s.logger.Info("starting gRPC server")
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return fmt.Errorf(
			"failed to listern tcp port %d, port might already be in use: %w",
			s.grpcPort,
			err,
		)
	}
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			grpclogging.UnaryServerInterceptor(InterceptorLogger(s.logger)),
		),
		grpc.ChainStreamInterceptor(
			grpclogging.StreamServerInterceptor(InterceptorLogger(s.logger)),
		),
	}
	if s.sslEnabled {
		opts = append(opts, grpc.Creds(s.grpcTlsConfig))
	} else {
		s.logger.Warn("running grpc server without tls!")
	}

	s.rpcServer = grpc.NewServer(opts...)

	for _, service := range s.services {
		service.Register(s.rpcServer)
	}

	go func() {
		err := s.rpcServer.Serve(lis)
		if err != nil {
			s.logger.Panic(
				fmt.Sprintf("failed to serve rpc server with tcp listener on port %d", s.grpcPort),
				zap.Error(err),
			)
		}
	}()
	return nil
}

func (s *Server) startHttpProxy() error {
	httpEndpoint := fmt.Sprintf(":%d", s.httpPort)
	grpcEndpoint := fmt.Sprintf(":%d", s.grpcPort)
	s.logger.Info("starting HTTP reverse proxy with RPC handler")
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(correlationIdMatcher),
		runtime.WithForwardResponseOption(correlationIdResponseModifier),
	)
	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			grpclogging.UnaryClientInterceptor(InterceptorLogger(s.logger)),
		),
		grpc.WithChainStreamInterceptor(
			grpclogging.StreamClientInterceptor(InterceptorLogger(s.logger)),
		),
	}
	if s.sslEnabled {
		s.logger.Info("configuring client tls credentials")
		opts = append(opts, grpc.WithTransportCredentials(s.httpProxyTlsConfig))
	} else {
		s.logger.Warn("dial options insecure")
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	for _, service := range s.services {
		err := service.RegisterProxy(s.ctx, mux, grpcEndpoint, opts)
		if err != nil {
			return fmt.Errorf(
				"unable to register service proxy handler on port %d: %w",
				s.httpPort,
				err,
			)
		}
	}
	err := s.healthService.RegisterProxy(s.ctx, mux, grpcEndpoint, opts)
	if err != nil {
		return fmt.Errorf("failed to register health service proxy handler: %w", err)
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	go func() {
		if s.sslEnabled {
			s.logger.Info("starting http server")
			err = http.ListenAndServeTLS(httpEndpoint, s.sslCertPath, s.sslKeyPath, mux)
			if err != nil {
				s.logger.Panic("failed to start http server with TLS configured", zap.Error(err))
			}
		} else {
			s.logger.Warn("start http server without TLS!")
			err = http.ListenAndServe(httpEndpoint, mux)
			if err != nil {
				s.logger.Panic("failed to start http server", zap.Error(err))
			}
		}
	}()
	return nil
}

func (s *Server) Stop() {
	s.healthService.SetHealth(false)
	s.logger.Info("set health endpoint to NOT_SERVING")

	// sleep to await stop ALB from sending requests to this instance
	time.Sleep(time.Duration(s.albDeregistrationSeconds) * time.Second)

	s.rpcServer.GracefulStop()
	s.logger.Info("stopped gRPC server gracefully")
}
