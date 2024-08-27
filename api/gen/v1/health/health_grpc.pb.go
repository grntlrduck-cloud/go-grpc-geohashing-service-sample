// original from https://github.com/grpc/grpc-proto/blob/master/grpc/health/v1/health.proto
// for the simple use case of this service with grpc-gateway it is just easier to implement a own health check

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             (unknown)
// source: v1/health/health.proto

package health

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.62.0 or later.
const _ = grpc.SupportPackageIsVersion8

const (
	HealthService_HealthCheck_FullMethodName = "/api.v1.health.HealthService/HealthCheck"
)

// HealthServiceClient is the client API for HealthService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Health is gRPC's mechanism for checking whether a server is able to handle
// RPCs. Its semantics are documented in
// https://github.com/grpc/grpc/blob/master/doc/health-checking.md.
type HealthServiceClient interface {
	// Check gets the health of the specified service. If the requested service
	// is unknown, the call will fail with status NOT_FOUND. If the caller does
	// not specify a service name, the server should respond with its overall
	// health status.
	//
	// Clients should set a deadline when calling Check, and can declare the
	// server unhealthy if they do not receive a timely response.
	//
	// Check implementations should be idempotent and side effect free.
	HealthCheck(ctx context.Context, in *HealthCheckRequest, opts ...grpc.CallOption) (*HealthCheckResponse, error)
}

type healthServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewHealthServiceClient(cc grpc.ClientConnInterface) HealthServiceClient {
	return &healthServiceClient{cc}
}

func (c *healthServiceClient) HealthCheck(ctx context.Context, in *HealthCheckRequest, opts ...grpc.CallOption) (*HealthCheckResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(HealthCheckResponse)
	err := c.cc.Invoke(ctx, HealthService_HealthCheck_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// HealthServiceServer is the server API for HealthService service.
// All implementations should embed UnimplementedHealthServiceServer
// for forward compatibility
//
// Health is gRPC's mechanism for checking whether a server is able to handle
// RPCs. Its semantics are documented in
// https://github.com/grpc/grpc/blob/master/doc/health-checking.md.
type HealthServiceServer interface {
	// Check gets the health of the specified service. If the requested service
	// is unknown, the call will fail with status NOT_FOUND. If the caller does
	// not specify a service name, the server should respond with its overall
	// health status.
	//
	// Clients should set a deadline when calling Check, and can declare the
	// server unhealthy if they do not receive a timely response.
	//
	// Check implementations should be idempotent and side effect free.
	HealthCheck(context.Context, *HealthCheckRequest) (*HealthCheckResponse, error)
}

// UnimplementedHealthServiceServer should be embedded to have forward compatible implementations.
type UnimplementedHealthServiceServer struct {
}

func (UnimplementedHealthServiceServer) HealthCheck(context.Context, *HealthCheckRequest) (*HealthCheckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HealthCheck not implemented")
}

// UnsafeHealthServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to HealthServiceServer will
// result in compilation errors.
type UnsafeHealthServiceServer interface {
	mustEmbedUnimplementedHealthServiceServer()
}

func RegisterHealthServiceServer(s grpc.ServiceRegistrar, srv HealthServiceServer) {
	s.RegisterService(&HealthService_ServiceDesc, srv)
}

func _HealthService_HealthCheck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HealthCheckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HealthServiceServer).HealthCheck(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HealthService_HealthCheck_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HealthServiceServer).HealthCheck(ctx, req.(*HealthCheckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// HealthService_ServiceDesc is the grpc.ServiceDesc for HealthService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var HealthService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.v1.health.HealthService",
	HandlerType: (*HealthServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "HealthCheck",
			Handler:    _HealthService_HealthCheck_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "v1/health/health.proto",
}
