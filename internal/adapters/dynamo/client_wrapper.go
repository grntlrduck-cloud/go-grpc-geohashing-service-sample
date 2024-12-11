package dynamo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DBClient interface {
	BatchPutItem(
		ctx context.Context,
		input *dynamodb.BatchWriteItemInput,
	) (*dynamodb.BatchWriteItemOutput, error)
	PutItem(ctx context.Context, input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	QueryItem(ctx context.Context, input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	CreateTable(
		ctx context.Context,
		input *dynamodb.CreateTableInput,
	) (*dynamodb.CreateTableOutput, error)
}

type ClientWrapper struct {
	dynamoClient *dynamodb.Client
	region       string
	override     bool
	hostOverride string
	portOverride string
	ctx          context.Context
}

type ClientOptions func(cw *ClientWrapper)

func WithRegion(region string) ClientOptions {
	return func(cw *ClientWrapper) {
		cw.region = region
	}
}

func WithEndPointOverride(host, port string) ClientOptions {
	return func(cw *ClientWrapper) {
		cw.override = true
		cw.hostOverride = host
		cw.portOverride = port
	}
}

func WithContext(ctx context.Context) ClientOptions {
	return func(cw *ClientWrapper) {
		cw.ctx = ctx
	}
}

func (client *ClientWrapper) BatchPutItem(
	ctx context.Context,
	input *dynamodb.BatchWriteItemInput,
) (*dynamodb.BatchWriteItemOutput, error) {
	output, err := client.dynamoClient.BatchWriteItem(ctx, input)
	return output, err
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

func (client *ClientWrapper) CreateTable(
	ctx context.Context,
	input *dynamodb.CreateTableInput,
) (*dynamodb.CreateTableOutput, error) {
	output, err := client.dynamoClient.CreateTable(ctx, input)
	return output, err
}

func NewClientWrapper(opts ...ClientOptions) (DBClient, error) {
	cw := &ClientWrapper{
		region: "eu-west-1",
		ctx:    context.Background(),
	}
	for _, opt := range opts {
		opt(cw)
	}
	awsOpts := []func(*config.LoadOptions) error{
		config.WithRegion(cw.region),
	}
	if cw.override {
		awsOpts = append(
			awsOpts,
			config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID: "test", SecretAccessKey: "test", SessionToken: "test",
					Source: "Mock credentials used above for local instance",
				},
			}),
		)
	}
	awsConfig, err := config.LoadDefaultConfig(cw.ctx, awsOpts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create aws config, are credentials configured? %w", err)
	}
	dyanmoOpts := []func(opt *dynamodb.Options){
		func(opt *dynamodb.Options) {
			opt.Region = cw.region
		},
	}
	if cw.override {
		dyanmoOpts = append(dyanmoOpts, func(options *dynamodb.Options) {
			options.BaseEndpoint = aws.String(
				fmt.Sprintf("http://%s:%s", cw.hostOverride, cw.portOverride),
			)
		})
	}
	dynamoClient := dynamodb.NewFromConfig(awsConfig, dyanmoOpts...)
	return &ClientWrapper{dynamoClient: dynamoClient}, nil
}
