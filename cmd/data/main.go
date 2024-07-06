package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func main() {
	ctx := context.Background()
	conf, confErr := config.LoadDefaultConfig(ctx)
	if confErr != nil {
		panic(fmt.Errorf("failed to load aws config: %w", confErr))
	}

	log.Print("getting bucket name parameter")
	ssmService := ssm.NewFromConfig(conf)
	bucketNameParam := "/config/go-grpc-poi-service/charging-data-bucket-name"
	bucketParam, getParamErr := ssmService.GetParameter(ctx, &ssm.GetParameterInput{
		Name: &(bucketNameParam),
	})
	if getParamErr != nil {
		panic(fmt.Errorf("failed to get ssm param: %w", getParamErr))
	}
	bucketName := bucketParam.Parameter.Value
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
	s3service := s3.NewFromConfig(conf)
	_, s3PutObjectErr := s3service.PutObject(ctx, &s3.PutObjectInput{
		Bucket: bucketName,
		Key:    &keyRaw,
		Body:   baseCsv,
	})
	if s3PutObjectErr != nil {
		panic(fmt.Errorf("failed to put csv %s to s3: %w", keyRaw, s3PutObjectErr))
	}
	log.Printf("uploaded csv to s3 with key %s", keyRaw)
}
