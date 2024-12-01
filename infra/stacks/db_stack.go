package stacks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"

	mycnstrcts "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/infra/constructs"
)

type DbStackProps struct {
	StackProps awscdk.StackProps
	TableName  string
	AppName    string
	LambdaPath string
}

type DbStack struct {
	awscdk.Stack
	Table awsdynamodb.ITable
}

func NewDbStack(
	scope constructs.Construct,
	id string,
	props *DbStackProps,
) *DbStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, jsii.String(id), &sprops)

	bucketParam := awsssm.StringParameter_FromStringParameterName(
		stack,
		jsii.String("BucketNameParam"),
		jsii.Sprintf("/config/%s/charging-data-bucket-name", props.AppName),
	)
	bucket := awss3.Bucket_FromBucketName(
		stack,
		jsii.String("DataBucket"),
		bucketParam.StringValue(),
	)

	csvObjectPath := "dynamo/csv/cpoi_dynamo_items.csv"

	tableProps := &awsdynamodb.TablePropsV2{
		TableName:     &props.TableName,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY, // don't in your company
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("pk"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		Billing: awsdynamodb.Billing_Provisioned(&awsdynamodb.ThroughputProps{
			WriteCapacity: awsdynamodb.Capacity_Autoscaled(
				&awsdynamodb.AutoscaledCapacityOptions{
					MinCapacity: jsii.Number(10.0),
					MaxCapacity: jsii.Number(50.0),
				},
			),
			ReadCapacity: awsdynamodb.Capacity_Autoscaled(
				&awsdynamodb.AutoscaledCapacityOptions{
					MinCapacity: jsii.Number(10.0),
					MaxCapacity: jsii.Number(50.0),
				},
			),
		}),
		GlobalSecondaryIndexes: &[]*awsdynamodb.GlobalSecondaryIndexPropsV2{
			{
				IndexName: jsii.String("gsi1_geo"),
				PartitionKey: &awsdynamodb.Attribute{
					Name: jsii.String("gsi1_geo_pk"),
					Type: awsdynamodb.AttributeType_NUMBER,
				},
				SortKey: &awsdynamodb.Attribute{
					Name: jsii.String("gsi1_geo_sk"),
					Type: awsdynamodb.AttributeType_NUMBER,
				},
				WriteCapacity: awsdynamodb.Capacity_Autoscaled(
					&awsdynamodb.AutoscaledCapacityOptions{
						MinCapacity: jsii.Number(10.0),
						MaxCapacity: jsii.Number(50.0),
					},
				),
				ReadCapacity: awsdynamodb.Capacity_Autoscaled(
					&awsdynamodb.AutoscaledCapacityOptions{
						MinCapacity: jsii.Number(10.0),
						MaxCapacity: jsii.Number(150.0),
					},
				),
			},
		},
	}

	tableWithInitPois := mycnstrcts.NewDynamoDBWithInitialData(
		stack,
		jsii.Sprintf("GeoIndexTable"),
		&mycnstrcts.DynamoDBWithInitialDataProps{
			TableProps:    tableProps,
			TableName:     props.TableName,
			Bucket:        bucket,
			CsvObjectPath: csvObjectPath,
			LambdaPath:    props.LambdaPath,
		},
	)
	return &DbStack{Stack: stack, Table: tableWithInitPois.Table}
}
