package constructs

import (
	"strconv"
	"time"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/customresources"
	"github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"
	awsconstructs "github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type DynamoDBWithInitialDataProps struct {
	TableName     string
	CsvObjectPath string
	Bucket        awss3.IBucket
	TableProps    *awsdynamodb.TablePropsV2
	LambdaPath    string
}

type DynamoDBWithInitialData struct {
	awsconstructs.Construct
	Table awsdynamodb.ITableV2
}

func NewDynamoDBWithInitialData(
	scope awsconstructs.Construct,
	id *string,
	props *DynamoDBWithInitialDataProps,
) *DynamoDBWithInitialData {
	construct := awsconstructs.NewConstruct(scope, id)

	table := awsdynamodb.NewTableV2(
		construct,
		jsii.String("DynamoDBTable"),
		props.TableProps,
	)

	lambdaEntryPath := "cmd/lambda"
	if props.LambdaPath != "" {
		lambdaEntryPath = props.LambdaPath
	}

	// create the lambda which will upload the poi items on create of CFN resource
	lambda := awscdklambdagoalpha.NewGoFunction(
		construct,
		jsii.String("OnCreateCfnDataInitLambda"),
		&awscdklambdagoalpha.GoFunctionProps{
			Architecture: awslambda.Architecture_ARM_64(),
			LogRetention: awslogs.RetentionDays_TWO_WEEKS,
			Entry:        &lambdaEntryPath,
			MemorySize:   jsii.Number(512),
			Timeout:      awscdk.Duration_Minutes(jsii.Number(15)),
			Environment: &map[string]*string{
				"TABLE_NAME":      &props.TableName,
				"BUCKET_NAME":     props.Bucket.BucketName(),
				"CSV_OBJECT_PATH": &props.CsvObjectPath,
			},
		},
	)
	table.GrantWriteData(lambda)
	props.Bucket.GrantRead(lambda, &props.CsvObjectPath)

	provider := customresources.NewProvider(
		construct,
		jsii.String("LoadDataProvider"),
		&customresources.ProviderProps{
			OnEventHandler: lambda,
		},
	)

	awscdk.NewCustomResource(construct, jsii.String("LoadDataTrigger"), &awscdk.CustomResourceProps{
		ServiceToken: provider.ServiceToken(),
		Properties: &map[string]any{
			"Timestamp":     jsii.String(strconv.FormatInt(time.Now().Unix(), 10)),
			"ResourceId":    jsii.String(props.TableName + "-data-loader"),
			"TableName":     jsii.String(props.TableName),
			"CsvObjectPath": jsii.String(props.CsvObjectPath),
		},
	})
	return &DynamoDBWithInitialData{
		Construct: construct,
		Table:     table,
	}
}
