package stacks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"

	mycnstrcts "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/infra/constructs"
)

type AppStackProps struct {
	StackProps awscdk.StackProps
	Table      awsdynamodb.ITable
	AppName    string
}

func NewAppStack(scope constructs.Construct, id string, props *AppStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	awsssm.NewStringParameter(stack, jsii.String("DummyResource"), &awsssm.StringParameterProps{
		StringValue: jsii.String("/config/app/dummy"),
	})

	mycnstrcts.LandingZoneVPC(stack, "LandingZoneDefaultVPC")
	mycnstrcts.LandingHostedZone(stack, "LandingZoneHosetedZone")

	// TODO: implement ALB balnced fargate service with own cluster and attach WAF to alb

	return stack
}
