package stacks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"

	"github.com/grntlduck-cloud/go-grpc-geohasing-service-sample/infra/utils"
)

type DataStackProps struct {
	StackProps awscdk.StackProps
	AppName    string
}

func NewDataStack(scope constructs.Construct, id string, props *DataStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	databucket := awss3.NewBucket(stack, jsii.String("ChargingDataBucket"), &awss3.BucketProps{
		EnforceSSL: jsii.Bool(true),
		Versioned:  jsii.Bool(true),
		LifecycleRules: &[]*awss3.LifecycleRule{
			{
				Enabled:                     jsii.Bool(true),
				NoncurrentVersionExpiration: awscdk.Duration_Days(jsii.Number(30)),
			},
		},
	})
  utils.OverrideLogicalId(databucket.Node(), "ChargingDataBucket")

  // params
	bucketNameParam := awsssm.NewStringParameter(
		stack,
		jsii.String("SSMChargingDataBucketName"),
		&awsssm.StringParameterProps{
			ParameterName: jsii.Sprintf("/config/%s/charging-data-bucket-name", props.AppName),
			StringValue:   databucket.BucketName(),
		},
	)
	utils.OverrideLogicalId(bucketNameParam.Node(), "SSMChargingDataBucketName")

  bucketArnParam := awsssm.NewStringParameter(
		stack,
		jsii.String("SSMChargingDataBucketArn"),
		&awsssm.StringParameterProps{
			StringValue:   databucket.BucketArn(),
			ParameterName: jsii.Sprintf("/config/%s/charging-data-bucket-arn", props.AppName),
		},
	)
  utils.OverrideLogicalId(bucketArnParam.Node(), "SSMChargingDataBucketArn")

	// outputs
  outputBucketName := awscdk.NewCfnOutput(stack, jsii.String("ChargingDataBucketNameOutPut"), &awscdk.CfnOutputProps{
		ExportName: jsii.Sprintf("%s-", "charging-data-bucket-name"),
		Value:      databucket.BucketName(),
	})
  outputBucketName.OverrideLogicalId(jsii.String("ChargingDataBucketNameOutPut"))

  outputBucketArn := awscdk.NewCfnOutput(stack, jsii.String("ChargingDataBucketArnOutPut"), &awscdk.CfnOutputProps{
		ExportName: jsii.Sprintf("%s-", "charging-data-bucket-arn"),
		Value:      databucket.BucketArn(),
	})
  outputBucketArn.OverrideLogicalId(jsii.String("ChargingDataBucketArnOutPut"))

	return stack
}
