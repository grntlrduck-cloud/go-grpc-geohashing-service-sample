package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gocarina/gocsv"
	"go.uber.org/zap"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/adapters/dynamo"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/domain/poi"
)

type TableInitHandler struct {
	logger        *zap.Logger
	repository    poi.Repository
	tableName     string
	bucketName    string
	csvObjectPath string
	s3Client      *s3.Client
}

func (handler *TableInitHandler) HandleCfn(ctx context.Context, event *cfn.Event) {
	resp := cfn.NewResponse(event)
	resourceID := event.ResourceProperties["ResourceId"]
	resourceIDStr := event.PhysicalResourceID
	if resourceID != nil {
		resourceIDStr = resourceID.(string)
		handler.logger.Info(
			"using resource id from properties",
			zap.String("resource_id", resourceIDStr),
		)
	}
	resp.PhysicalResourceID = resourceIDStr

	logger := handler.logger.With(
		zap.String("request_id", event.RequestID),
		zap.String("resource_id", resourceIDStr),
		zap.String("event_type", string(event.RequestType)),
		zap.String("bucket_name", handler.bucketName),
		zap.String("data_object_key", handler.csvObjectPath),
	)

	if event.RequestType != cfn.RequestCreate {
		logger.Info("No create event, retuening instant with success")
		resp.Status = cfn.StatusSuccess
		_ = resp.Send()
		return
	}
	logger.Info("Processing CREATE event, initiating table")

	// get data from s3
	items, err := handler.getDataFromS3(ctx)
	if err != nil {
		logger.Error(
			"failed to read data from s3",
			zap.Error(err),
		)
		resp.Status = cfn.StatusFailed
		_ = resp.Send()
	}

	// map items to domain
	domain := make([]*poi.PoILocation, len(items))
	for i, v := range items {
		d, errD := v.Domain()
		if errD != nil {
			logger.Error(
				"failed to map items to domain",
				zap.Error(errD),
			)
			resp.Status = cfn.StatusFailed
			_ = resp.Send()
		}
		domain[i] = d
	}

	// do batch upsert off all
	err = handler.repository.UpsertBatch(ctx, domain, handler.logger)
	if err != nil {
		handler.logger.Error("failed to upsert batches to table", zap.Error(err))
		resp.Status = cfn.StatusFailed
		_ = resp.Send()
		return
	}

	// send success response to CFN
	resp.Status = cfn.StatusSuccess
	_ = resp.Send()
	logger.Info("Successful initiated table with data. Done!")
}

func (handler *TableInitHandler) getDataFromS3(ctx context.Context) ([]*dynamo.CPoIItem, error) {
	data, err := handler.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &handler.bucketName,
		Key:    &handler.csvObjectPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from s3: %w", err)
	}
	defer data.Body.Close()
	var items []*dynamo.CPoIItem
	err = gocsv.Unmarshal(data.Body, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to map data to struct: %w", err)
	}
	return items, nil
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// get env vars for table, bucket and object path
	tableName := os.Getenv("TABLE_NAME")
	bucketName := os.Getenv("BUCKET_NAME")
	csvObjectPath := os.Getenv("CSV_OBJECT_PATH")

	if tableName == "" || bucketName == "" || csvObjectPath == "" {
		logger.Panic(
			"failed to resolve environment",
			zap.String("table_name", tableName),
			zap.String("bucket_name", bucketName),
			zap.String("csv_object_path", csvObjectPath),
		)
	}

	// create dynamoClient and s3 client
	ctx := context.Background()
	dynamoClient, err := dynamo.NewClientWrapper(dynamo.WithContext(ctx))
	if err != nil {
		logger.Panic("failed to init dynamodb client")
	}
	conf, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Panic("failed to load aws config", zap.Error(err))
	}
	s3Client := s3.NewFromConfig(conf)

	// create the repo
	repo, err := dynamo.NewPoIGeoRepository(
		logger,
		dynamo.WithDynamoClientWrapper(dynamoClient),
		dynamo.WithTableName(tableName),
		dynamo.WithCreateAndInitTable(false),
	)
	if err != nil {
		logger.Panic("failed to init repository", zap.Error(err))
	}

	handler := &TableInitHandler{
		logger:        logger,
		repository:    repo,
		tableName:     tableName,
		bucketName:    bucketName,
		csvObjectPath: csvObjectPath,
		s3Client:      s3Client,
	}

	lambda.Start(handler.HandleCfn)
}
