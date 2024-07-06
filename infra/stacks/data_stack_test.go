package stacks_test

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
	. "github.com/onsi/ginkgo/v2"

	"github.com/grntlduck-cloud/go-grpc-geohasing-service-sample/infra/stacks"
)

var _ = Describe("Given app", func() {
	app := awscdk.NewApp(nil)

	When("has data stack", func() {
		stack := stacks.NewDataStack(app, "test-data-stack", &stacks.DataStackProps{
			StackProps: awscdk.StackProps{
				Env: &awscdk.Environment{
					Account: jsii.String("123456789012"),
					Region:  jsii.String("eu-west-1"),
				},
			},
			AppName: "test",
		})
		template := assertions.Template_FromStack(stack, nil)

		It("has data bucket", func() {
			template.ResourceCountIs(jsii.String("AWS::S3::Bucket"), jsii.Number(1))
		})

		It("has ssm parameter with bucket name", func() {
			template.HasResourceProperties(jsii.String("AWS::SSM::Parameter"), map[string]any{
				"Name": jsii.String("/config/test/charging-data-bucket-arn"),
				"Value": map[string][]string{
					"Fn::GetAtt": {"ChargingDataBucket", "Arn"},
				},
			})
		})

		It("has ssm parameter with bucket arn", func() {
			template.HasResourceProperties(jsii.String("AWS::SSM::Parameter"), map[string]any{
				"Name": jsii.String("/config/test/charging-data-bucket-name"),
				"Value": map[string]string{
					"Ref": "ChargingDataBucket",
				},
			})
		})

		It("outputs keep the same export name", func() {
			template.HasOutput(jsii.String("ChargingDataBucketNameOutPut"), map[string]any{
				"Export": map[string]string{
					"Name": "test-data-stack-charging-data-bucket-name",
				},
			})

			template.HasOutput(jsii.String("ChargingDataBucketArnOutPut"), map[string]any{
				"Export": map[string]string{
					"Name": "test-data-stack-charging-data-bucket-arn",
				},
			})
		})
	})
})
