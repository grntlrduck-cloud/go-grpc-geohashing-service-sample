package test

import (
	"context"
	"fmt"

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
	Expect(resp).To(Not(BeNil()))
	return resp
}

func NewHealthRpcClient(port int32) *HealthRpcClient {
	adr := fmt.Sprintf("localhost:%d", port)
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	client, err := grpc.NewClient(adr, dialOpts...)
	Expect(err).To(Not(HaveOccurred()))
	healthClient := health.NewHealthServiceClient(client)
	Expect(healthClient).To(Not(BeNil()))
	return &HealthRpcClient{client: healthClient}
}
