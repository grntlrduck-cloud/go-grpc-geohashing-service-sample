package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/amazon-ion/ion-go/ion"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/gocarina/gocsv"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/dynamo"
)

const (
	bucketNameParam        = "/config/go-grpc-poi-service/charging-data-bucket-name"
	cPoIDataCSVPath        = "cpoi_data.csv"
	cPoIDynamoItemsCSVPath = "cpoi_dynamo_items.csv"
	cPoIIonFilePath        = "cpoi_ion_items"
)

// This program requires the dataset from kaggle to be present in the root of this project as 'cpoi_data.csv'.
// The CSV is proecessed and mapped to fit the data model for dynamo db, saved to disk as CSV and AWS ION.
// Finally, the files, the raw and the processed data is uploaded to the S3 bucket defined in the data-stack
// of the infrasturcture.
// TODO: This program could be optimized by using go flags to conrtol processing, uploads, and file name cusotmization
func main() {
	ctx := context.Background()
	conf, confErr := config.LoadDefaultConfig(ctx)
	if confErr != nil {
		panic(fmt.Errorf("failed to load aws config: %w", confErr))
	}

	log.Print("getting bucket name parameter")
	ssmClient := ssm.NewFromConfig(conf)
	bucketName := paramStr(ctx, ssmClient, bucketNameParam)
	log.Printf("got bucket name from parameter with value %s", *bucketName)

	// load the csv from file
	baseCsv := loadFile(cPoIDataCSVPath)
	defer baseCsv.Close()

	// map entries to struct
	log.Print("mapping data to dynamoItems")
	entries := []*dynamo.ChargingCSVEntry{}
	if csvUnmarshallErr := gocsv.UnmarshalFile(baseCsv, &entries); csvUnmarshallErr != nil {
		panic(fmt.Errorf("failed to map rows to struct, %w", csvUnmarshallErr))
	}
	dynamoItems := dynamo.EntriesToDynamo(entries)

	// write the dyanmoItems as csv to file
	log.Print("writing dynamo items csv to files")
	writeCSV(dynamoItems, cPoIDynamoItemsCSVPath)

	log.Print("writing ion file")
	writeIonFile(dynamoItems, cPoIIonFilePath)

	log.Print("uploading CSVs to s3")
	// upload the raw data
	s3Client := s3.NewFromConfig(conf)
	baseCsvS3 := loadFile(cPoIDataCSVPath)
	defer baseCsvS3.Close()
	putObject(ctx, s3Client, aws.String("raw/raw_cpoi_data.csv"), bucketName, baseCsvS3)

	// upload the processed csv
	dynamoItemsCsvS3 := loadFile(cPoIDynamoItemsCSVPath)
	defer dynamoItemsCsvS3.Close()
	putObject(
		ctx,
		s3Client,
		aws.String("dynamo/csv/cpoi_dynamo_items.csv"),
		bucketName,
		dynamoItemsCsvS3,
	)

	// upload the ion items
	ionFile := loadFile(cPoIIonFilePath)
	defer ionFile.Close()
	putObject(
		ctx,
		s3Client,
		aws.String("dynamo/ion/cpoi_ion_items"),
		bucketName,
		ionFile,
	)

	log.Print("Done!")
}

func writeCSV(items []*dynamo.CPoIItem, filePath string) {
	itemsFile, createFileErr := os.OpenFile(
		filePath,
		os.O_CREATE|os.O_WRONLY,
		0644,
	)
	defer closeFile(itemsFile)
	if createFileErr != nil {
		panic(fmt.Errorf("failed to create file for dynamo items, %w", createFileErr))
	}
	csvMarshalErr := gocsv.MarshalFile(items, itemsFile)
	if csvMarshalErr != nil {
		panic(fmt.Errorf("failed to marshal dynamo items to csv, %w", csvMarshalErr))
	}
}

func writeIonFile(items []*dynamo.CPoIItem, filePath string) {
	ionF, ionFileErr := os.OpenFile(
		filePath,
		os.O_CREATE|os.O_WRONLY,
		0644,
	)
	defer closeFile(ionF)
	if ionFileErr != nil {
		panic(fmt.Errorf("failed to write ion file, %w", ionFileErr))
	}
	writer := ion.NewTextWriter(ionF)
  defer func(w ion.Writer) {
    e := w.Finish()
    if e != nil {
      panic(fmt.Errorf("failed to close file, %w", e))
    }
  }(writer)
	encoder := ion.NewEncoder(writer)
	for _, v := range items {
		ion := v.IonItem()
		e := encoder.Encode(ion)
		if e != nil {
			panic(fmt.Errorf("failed to encode ion itemm, %w", e))
		}
	}
}

func closeFile(f *os.File) {
  e := f.Close()
  if e != nil {
    panic(fmt.Errorf("failed to close file, %w", e))
  }
}

func loadFile(filePath string) *os.File {
	csv, fileErr := os.Open(filePath)
	if fileErr != nil {
		panic(fmt.Errorf("failed to load csv from file: %w", fileErr))
	}
	return csv
}

func paramStr(ctx context.Context, ssmClient *ssm.Client, paramName string) *string {
	param, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: &(paramName),
	})
	if err != nil {
		panic(fmt.Errorf("faile to get param, %w", err))
	}
	return param.Parameter.Value
}

func putObject(
	ctx context.Context,
	s3Client *s3.Client,
	key *string,
	bucket *string,
	body io.Reader,
) {
	_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: bucket,
		Key:    key,
		Body:   body,
	})
	if err != nil {
		panic(fmt.Errorf("failed to put object %s to s3, %w", *key, err))
	}
}
