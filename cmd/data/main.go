package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	_ "github.com/gocarina/gocsv" // TODO use, impelement parsing of csv data to our data model for DynamoDB
)

func main() {
	ctx := context.Background()
	conf, confErr := config.LoadDefaultConfig(ctx)
	if confErr != nil {
		panic(fmt.Errorf("failed to load aws config: %w", confErr))
	}

	log.Print("getting bucket name parameter")
	ssmClient := ssm.NewFromConfig(conf)
	bucketNameParam := "/config/go-grpc-poi-service/charging-data-bucket-name"
	bucketName, getParamErr := paramStr(ctx, ssmClient, bucketNameParam)
	if getParamErr != nil {
		panic(fmt.Errorf("failed to get ssm param: %w", getParamErr))
	}
	log.Printf("got bucket name from parameter with value %s", *bucketName)

	// load the csv from file
	log.Print("loading csv from file...")
	baseCsv, fileErr := os.Open("cpoi_data.csv")
	if fileErr != nil {
		panic(fmt.Errorf("failed to load csv from file: %w", fileErr))
	}
	defer baseCsv.Close()

	// put base data to s3
	log.Print("uploading uprocessed csv to s3...")
	keyRaw := "raw_cpoi_data.csv"
	s3Client := s3.NewFromConfig(conf)
	s3PutObjectErr := putObject(ctx, s3Client, &keyRaw, bucketName, baseCsv)
	if s3PutObjectErr != nil {
		panic(fmt.Errorf("failed to put csv %s to s3: %w", keyRaw, s3PutObjectErr))
	}
	log.Printf("uploaded csv to s3 with key %s", keyRaw)
}

func paramStr(ctx context.Context, ssmClient *ssm.Client, paramName string) (*string, error) {
	param, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: &(paramName),
	})
	if err != nil {
		return nil, err
	}
	return param.Parameter.Value, nil
}

func putObject(
	ctx context.Context,
	s3Client *s3.Client,
	key *string,
	bucket *string,
	body io.Reader,
) error {
	_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: bucket,
		Key:    key,
		Body:   body,
	})
	if err != nil {
		return err
	}
	return nil
}
