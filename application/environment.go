package application

import (
	"os"
	"errors"
)

type ServiceEnvironment struct {
	TableName string
	Region    string
}

func NewServiceEnv() *ServiceEnvironment {
	tableName := os.Getenv("DYNAMO_TABLE_NAME")
	if tableName == "" {
		panic(errors.New("DYNAMO_TABLE_NAME is not set"))
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-west-1"
	}
	return &ServiceEnvironment{TableName: tableName, Region: region}
}
