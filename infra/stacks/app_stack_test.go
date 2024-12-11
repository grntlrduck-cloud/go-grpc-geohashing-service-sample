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
		dbStack := stacks.NewDBStack(app, "test-db-stack", &stacks.DBStackProps{
			StackProps: awscdk.StackProps{
				Env: &awscdk.Environment{
					Account: jsii.String("123456789012"),
					Region:  jsii.String("eu-west-1"),
				},
			},
			AppName:    "test",
			LambdaPath: "../../cmd/lambda/",
		})
		stack := stacks.NewAppStack(app, "test-app-stack", &stacks.AppStackProps{
			StackProps: awscdk.StackProps{
				Env: &awscdk.Environment{
					Account: jsii.String("123456789012"),
					Region:  jsii.String("eu-west-1"),
				},
			},
			AppName: "test",
			Table:   dbStack.Table,
		})
		template = assertions.Template_FromStack(stack, nil)
	})

	When("stack template", func() {
		It("has secret", func() {
			template.ResourceCountIs(jsii.String("AWS::SecretsManager::Secret"), jsii.Number(1))
		})

		It("has alb", func() {
			template.ResourceCountIs(
				jsii.String("AWS::ElasticLoadBalancingV2::LoadBalancer"),
				jsii.Number(1),
			)
		})

		It("has listeners", func() {
			template.ResourceCountIs(
				jsii.String("AWS::ElasticLoadBalancingV2::ListenerRule"),
				jsii.Number(2),
			)
		})

		It("has listeners", func() {
			template.ResourceCountIs(
				jsii.String("AWS::ElasticLoadBalancingV2::TargetGroup"),
				jsii.Number(2),
			)
		})

		It("has certificate", func() {
			template.ResourceCountIs(
				jsii.String("AWS::CertificateManager::Certificate"),
				jsii.Number(1),
			)
		})

		It("has record set", func() {
			template.ResourceCountIs(jsii.String("AWS::Route53::RecordSet"), jsii.Number(1))
		})

		It("has web acl", func() {
			template.ResourceCountIs(jsii.String("AWS::WAFv2::WebACL"), jsii.Number(1))
		})

		It("has web acl association", func() {
			template.ResourceCountIs(jsii.String("AWS::WAFv2::WebACLAssociation"), jsii.Number(1))
		})

		It("has security groups", func() {
			template.ResourceCountIs(jsii.String("AWS::EC2::SecurityGroup"), jsii.Number(2))
		})

		It("has ecs cluster", func() {
			template.ResourceCountIs(jsii.String("AWS::ECS::Cluster"), jsii.Number(1))
		})

		It("has ecs service", func() {
			template.ResourceCountIs(jsii.String("AWS::ECS::Service"), jsii.Number(1))
		})
	})
})
