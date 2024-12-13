package stacks_test

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
	. "github.com/onsi/ginkgo/v2"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/infra/stacks"
)

var _ = Describe("Given app stack", Ordered, func() {
	var template assertions.Template
	BeforeAll(func() {
		app := awscdk.NewApp(nil)

		stack := stacks.NewDBStack(app, "test-db-stack", &stacks.DBStackProps{
			StackProps: awscdk.StackProps{
				Env: &awscdk.Environment{
					Account: jsii.String("123456789012"),
					Region:  jsii.String("eu-west-1"),
				},
			},
			AppName:    "test",
			LambdaPath: "../../cmd/lambda",
		})
		template = assertions.Template_FromStack(stack.Stack, nil)
	})

	When("stack template", func() {
		It("has table", func() {
			template.ResourceCountIs(
				jsii.String("AWS::DynamoDB::GlobalTable"),
				jsii.Number(1),
			)
		})

		It("has custom resource", func() {
			template.ResourceCountIs(
				jsii.String("AWS::CloudFormation::CustomResource"),
				jsii.Number(1),
			)
		})
	})
})
