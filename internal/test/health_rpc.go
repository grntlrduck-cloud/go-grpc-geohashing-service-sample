package test

import (
	"context"
	"fmt"

	. "github.com/onsi/gomega" //nolint:stylecheck
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/health"
)

type HealthRPCClient struct {
	client health.HealthServiceClient
}

func (h HealthRPCClient) CheckHealth() *health.HealthCheckResponse {
	resp, err := h.client.HealthCheck(
		context.Background(),
		&health.HealthCheckRequest{Service: ""},
	)
	Expect(err).To(Not(HaveOccurred()))
	Expect(resp).To(Not(BeNil()))
	return resp
}

func NewHealthRPCClient(port int32) *HealthRPCClient {
	adr := fmt.Sprintf("localhost:%d", port)
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	client, err := grpc.NewClient(adr, dialOpts...)
	Expect(err).To(Not(HaveOccurred()))
	healthClient := health.NewHealthServiceClient(client)
	Expect(healthClient).To(Not(BeNil()))
	return &HealthRPCClient{client: healthClient}
}
