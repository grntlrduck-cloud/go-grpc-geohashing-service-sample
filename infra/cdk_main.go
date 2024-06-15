package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/grntlduck-cloud/go-grpc-geohasing-service-sample/infra/stacks"
)

const appName = "go-grpc-geohashing-service-sample"

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	stacks.NewAppStack(app, fmt.Sprintf("%s-app-stack", appName), &stacks.AppStackProps{
		StackProps: awscdk.StackProps{
			Env: env(),
		},
		AppName: appName,
	})

	app.Synth(nil)
}

func env() *awscdk.Environment {
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("AWS_ACCOUNT")),
		Region:  jsii.String(os.Getenv("AWS_REGION")),
	}
}
