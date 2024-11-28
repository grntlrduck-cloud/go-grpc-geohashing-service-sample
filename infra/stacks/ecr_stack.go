package stacks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type EcrStackProps struct {
	StackProps awscdk.StackProps
	AppName    string
}

func NewEcrStack(scope constructs.Construct, id string, props *EcrStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, jsii.String(id), &sprops)

	// create ECR with reoval destroy and max image count
	serviceImageRepo := awsecr.NewRepository(
		stack,
		jsii.Sprintf("ECR-%s", props.AppName),
		&awsecr.RepositoryProps{
			RemovalPolicy: awscdk.RemovalPolicy_DESTROY, // don't do that in your company
			LifecycleRules: &[]*awsecr.LifecycleRule{
				{
					MaxImageCount: jsii.Number(15),
				},
			},
			ImageScanOnPush:    jsii.Bool(true),
			ImageTagMutability: awsecr.TagMutability_IMMUTABLE,
		},
	)

	// create SSM parameter to be resolve the ecr URI in app-stack
	awsssm.NewStringParameter(
		stack,
		jsii.Sprintf("ECRUriStringPramaeter-%s", props.AppName),
		&awsssm.StringParameterProps{
			ParameterName: jsii.Sprintf("/config/%s/ecr/uri", props.AppName),
			StringValue:   serviceImageRepo.RepositoryUri(),
		},
	)

	awsssm.NewStringParameter(
		stack,
		jsii.Sprintf("ECRNameStringPramaeter-%s", props.AppName),
		&awsssm.StringParameterProps{
			ParameterName: jsii.Sprintf("/config/%s/ecr/name", props.AppName),
			StringValue:   serviceImageRepo.RepositoryName(),
		},
	)

	awsssm.NewStringParameter(
		stack,
		jsii.Sprintf("ECRArnStringPramaeter-%s", props.AppName),
		&awsssm.StringParameterProps{
			ParameterName: jsii.Sprintf("/config/%s/ecr/arn", props.AppName),
			StringValue:   serviceImageRepo.RepositoryArn(),
		},
	)

	return stack
}
