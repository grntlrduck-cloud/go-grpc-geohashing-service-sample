package test

import (
	"context"
	"time"

	"github.com/docker/go-connections/nat"
	. "github.com/onsi/gomega"
	dynamodblocal "github.com/testcontainers/testcontainers-go/modules/dynamodb"
)

const (
	portProtocol nat.Port = "8000/tcp"
	dynamoImage  string   = "public.ecr.aws/aws-dynamodb-local/aws-dynamodb-local:2.5.3"
)

type DynamoContainer struct {
	ctx       context.Context
	host      string
	port      string
	container *dynamodblocal.DynamoDBContainer
}

func (d *DynamoContainer) Host() string {
	return d.port
}

func (d *DynamoContainer) Port() string {
	return d.host
}

func (d *DynamoContainer) Stop() {
	until := time.Duration(5 * time.Second)
	_ = d.container.Stop(d.ctx, &until)
}

func NewDynamoContainer(ctx context.Context) *DynamoContainer {
	dynamodbContainer, err := dynamodblocal.Run(
		ctx,
		dynamoImage,
	)
	Expect(err).To(Not(HaveOccurred()))
	host, err := dynamodbContainer.Host(ctx)
	Expect(err).To(Not(HaveOccurred()))
	port, err := dynamodbContainer.MappedPort(ctx, portProtocol)
	Expect(err).To(Not(HaveOccurred()))
	return &DynamoContainer{
		ctx:       ctx,
		host:      host,
		port:      port.Port(),
		container: dynamodbContainer,
	}
}
