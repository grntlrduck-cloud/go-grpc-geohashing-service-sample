package test

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	poiv1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/poi"
)

type PoIRpcClient struct {
	client poiv1.PoIServiceClient
}

func (p *PoIRpcClient) PoI(id string, correlation bool) (*poiv1.PoIResponse, error) {
	ctx := contextWithCorrelationId(correlation)
	resp, err := p.client.PoI(ctx, &poiv1.PoIRequest{Id: id})
	return resp, err
}

func (p *PoIRpcClient) Bbox(
	ne, sw *poiv1.Coordinate,
	correlation bool,
) (*poiv1.PoISearchResponse, error) {
	ctx := contextWithCorrelationId(correlation)
	resp, err := p.client.BBox(ctx, &poiv1.BBoxRequest{Bbox: &poiv1.BBox{Ne: ne, Sw: sw}})
	return resp, err
}

func (p *PoIRpcClient) Proximity(
	cntr *poiv1.Coordinate,
	radiusMeters float64,
	correlation bool,
) (*poiv1.PoISearchResponse, error) {
	ctx := contextWithCorrelationId(correlation)
	resp, err := p.client.Proximity(
		ctx,
		&poiv1.ProximityRequest{Center: cntr, RadiusMeters: radiusMeters},
	)
	return resp, err
}

func (p *PoIRpcClient) Route(
	route []*poiv1.Coordinate,
	correlation bool,
) (*poiv1.PoISearchResponse, error) {
	ctx := contextWithCorrelationId(correlation)
	resp, err := p.client.Route(ctx, &poiv1.RouteRequest{Route: route})
	return resp, err
}

func contextWithCorrelationId(correlation bool) context.Context {
	ctx := context.Background()
	if correlation {
		md := metadata.Pairs("X-Correlation-Id", uuid.NewString())
		return metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
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
