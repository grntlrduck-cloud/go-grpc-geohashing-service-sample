package test

import (
	"context"

	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/health"
)

type HealthRpcClient struct {
	client health.HealthServiceClient
}

func (h HealthRpcClient) AssertCheckHealth() *health.HealthCheckResponse {
	resp, err := h.client.HealthCheck(
		context.Background(),
		&health.HealthCheckRequest{Service: ""},
	)
	Expect(err).To(Not(HaveOccurred()))
	return resp
}

func NewHealthRpcClient() *HealthRpcClient {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	client, err := grpc.NewClient("localhost:7443", dialOpts...)
	Expect(err).To(Not(HaveOccurred()))
	healthClient := health.NewHealthServiceClient(client)
	Expect(healthClient).To(Not(BeNil()))
	return &HealthRpcClient{client: healthClient}
}
