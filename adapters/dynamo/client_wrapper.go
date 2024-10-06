package dynamo

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DbClient interface {
	BatchPutItem(
		ctx context.Context,
		input *dynamodb.BatchWriteItemInput,
	) (*dynamodb.BatchWriteItemOutput, error)
	PutItem(ctx context.Context, input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	QueryItem(ctx context.Context, input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
}

type ClientWrapper struct {
	dynamoClient *dynamodb.Client
}

func (client *ClientWrapper) PutItem(
	ctx context.Context,
	input *dynamodb.PutItemInput,
) (*dynamodb.PutItemOutput, error) {
	output, err := client.dynamoClient.PutItem(ctx, input)
	return output, err
}

func (client *ClientWrapper) GetItem(
	ctx context.Context,
	input *dynamodb.GetItemInput,
) (*dynamodb.GetItemOutput, error) {
	output, err := client.dynamoClient.GetItem(ctx, input)
	return output, err
}

func (client *ClientWrapper) QueryItem(
	ctx context.Context,
	input *dynamodb.QueryInput,
) (*dynamodb.QueryOutput, error) {
	output, err := client.dynamoClient.Query(ctx, input)
	return output, err
}

func NewClientWrapper(ctx context.Context, region string) *ClientWrapper {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	dynamoClient := dynamodb.NewFromConfig(awsConfig, func(opt *dynamodb.Options) {
		opt.Region = region
	})
	return &ClientWrapper{dynamoClient: dynamoClient}
}

// used for testing where the client points to local host
func NewFromClient(client *dynamodb.Client) *ClientWrapper {
	if client == nil {
		panic("client is nil")
	}
	return &ClientWrapper{dynamoClient: client}
}
