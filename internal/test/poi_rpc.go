package test

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/segmentio/ksuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	poiv1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/poi"
)

// TODO: implement other methods for testing
type PoIRpcClient struct {
	client poiv1.PoIServiceClient
}

func (p *PoIRpcClient) AssertPoI(id ksuid.KSUID) *poiv1.PoIResponse {
	ctx := p.contextWithCorrelationId()
	resp, err := p.client.PoI(ctx, &poiv1.PoIRequest{Id: id.String()})
	Expect(err).To(Not(HaveOccurred()))
	Expect(resp).To(Not(BeNil()))
	return resp
}

func (p *PoIRpcClient) contextWithCorrelationId() context.Context {
	md := metadata.Pairs("X-Correlation-Id", uuid.NewString())
	return metadata.NewOutgoingContext(context.Background(), md)
}

func NewPoIRpcClient(port int32) *PoIRpcClient {
	adr := fmt.Sprintf("localhost:%d", port)
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	client, err := grpc.NewClient(adr, dialOpts...)
	Expect(err).To(Not(HaveOccurred()))
	poiClient := poiv1.NewPoIServiceClient(client)
	return &PoIRpcClient{client: poiClient}
}
