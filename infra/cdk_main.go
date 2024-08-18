package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/infra/stacks"
)

const appName = "go-grpc-poi-service"

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	stacks.NewDataStack(app, fmt.Sprintf("%s-data-stack", appName), &stacks.DataStackProps{
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
