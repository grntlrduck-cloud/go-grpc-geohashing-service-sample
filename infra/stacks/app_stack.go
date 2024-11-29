package stacks

import (
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecspatterns"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
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

	imageTag := os.Getenv("GITHUB_SHA")
	if imageTag == "" {
		imageTag = "no-tag"
	}

	ecrArn := awsssm.StringParameter_FromStringParameterName(
		stack,
		jsii.String("EcrArnParam"),
		jsii.Sprintf("/config/%s/ecr/name", props.AppName),
	)
	ecrRepo := awsecr.Repository_FromRepositoryName(stack, jsii.String("ECR"), ecrArn.StringValue())

	vpc := mycnstrcts.LandingZoneVPC(stack, "LandingZoneDefaultVPC")
	hostedZone := mycnstrcts.LandingHostedZone(stack, "LandingZoneHosetedZone")

	// the API key based auth is not recommended for corporate environments,
	// but if you use it, rotate the secret at least every 30 days
	apiKeySecret := awssecretsmanager.NewSecret(
		stack,
		jsii.String("APIKeySecret"),
		&awssecretsmanager.SecretProps{
			GenerateSecretString: &awssecretsmanager.SecretStringGenerator{
				ExcludeCharacters: jsii.String(`"'\@/\\`), // exclude escape characters
			},
		},
	)

	containerName := fmt.Sprintf("%s-container", props.AppName)
	restProxyPort := 8443

	logGroup := awslogs.NewLogGroup(
		stack,
		jsii.String("ServiceLogGroup"),
		&awslogs.LogGroupProps{
			RemovalPolicy: awscdk.RemovalPolicy_DESTROY, // don't do in corporate area
			Retention:     awslogs.RetentionDays_TWO_WEEKS,
		},
	)

	service := awsecspatterns.NewApplicationLoadBalancedFargateService(
		stack,
		jsii.Sprintf("ALBFargateService-%s", props.AppName),
		&awsecspatterns.ApplicationLoadBalancedFargateServiceProps{
			ServiceName: &props.AppName,
			TaskImageOptions: &awsecspatterns.ApplicationLoadBalancedTaskImageOptions{
				ContainerName: &containerName,
				Image:         awsecs.ContainerImage_FromEcrRepository(ecrRepo, &imageTag),
				Secrets: &map[string]awsecs.Secret{
					"API_KEY_SECRET_VALUE": awsecs.Secret_FromSecretsManager(apiKeySecret, nil),
				},
				Environment: &map[string]*string{
					"APP_NAME":            &props.AppName,
					"APP_ENV":             jsii.String("prod"),
					"BOOT_PROFILE_ACTIVE": jsii.String("prod"),
					"ACCOUNT_ID":          props.StackProps.Env.Account,
				},
				ContainerPort: jsii.Number(443),
				LogDriver: awsecs.AwsLogDriver_AwsLogs(
					&awsecs.AwsLogDriverProps{
						LogGroup:     logGroup,
						StreamPrefix: jsii.Sprintf("FargateService-%s", containerName),
					},
				),
			},
			DesiredCount:    jsii.Number(1),
			Cpu:             jsii.Number(512),
			MemoryLimitMiB:  jsii.Number(1024),
			ListenerPort:    jsii.Number(443),
			TargetProtocol:  awselasticloadbalancingv2.ApplicationProtocol_HTTPS,
			ProtocolVersion: awselasticloadbalancingv2.ApplicationProtocolVersion_GRPC,
			RedirectHTTP:    jsii.Bool(true),
			Protocol:        awselasticloadbalancingv2.ApplicationProtocol_HTTPS,
			SslPolicy:       awselasticloadbalancingv2.SslPolicy_FORWARD_SECRECY_TLS12_RES_GCM,
			// ensure rolling update of containers
			MinHealthyPercent: jsii.Number(100),
			MaxHealthyPercent: jsii.Number(200),
			HealthCheck: &awsecs.HealthCheck{
				Command:     jsii.Strings("CMD-SHELL", "/service/probe"),
				StartPeriod: awscdk.Duration_Seconds(jsii.Number(10)),
				Interval:    awscdk.Duration_Seconds(jsii.Number(5)),
				Timeout:     awscdk.Duration_Seconds(jsii.Number(2)),
				Retries:     jsii.Number(3),
			},
			// ensure rollback using cercuit breaker
			CircuitBreaker: &awsecs.DeploymentCircuitBreaker{
				Enable:   jsii.Bool(true),
				Rollback: jsii.Bool(true),
			},
			DomainName: jsii.Sprintf("%s.%s", props.AppName, *hostedZone.ZoneName()),
			DomainZone: hostedZone,
			Vpc:        vpc,
			TaskSubnets: &awsec2.SubnetSelection{
				Subnets: vpc.PrivateSubnets(),
			},
		},
	)
	// add the port of our HTTP REST reverse proxy to the container port mpaaings
	for _, container := range *service.TaskDefinition().Containers() {
		if *container.ContainerName() == containerName {
			container.AddPortMappings(&awsecs.PortMapping{
				ContainerPort: jsii.Number(restProxyPort),
			})
		}
	}
	// grant read and write permissions to dynamo table
	props.Table.GrantWriteData(service.Service().TaskDefinition().TaskRole())

	// THE ALB AND ROUTING CONFIGURATIONS DOWN BELOW
	// ensure default action on listener
	service.Listener().
		AddAction(jsii.String("DefaultAction"),
			&awselasticloadbalancingv2.AddApplicationActionProps{
				Action: awselasticloadbalancingv2.ListenerAction_FixedResponse(
					jsii.Number(404),
					nil,
				),
			},
		)
	awscdk.Annotations_Of(service).AcknowledgeWarning(
		jsii.String("@aws-cdk/aws-elbv2:listenerExistingDefaultActionReplaced"),
		jsii.String("Default action intentionally set to 404"),
	)

	// modify l3 constructs target group and add health and actions
	targetGroup := service.TargetGroup()
	targetGroup.ConfigureHealthCheck(&awselasticloadbalancingv2.HealthCheck{
		Enabled: jsii.Bool(true),
		Path:    jsii.String("/api.v1.health.HealthService/HealthCheck"),
		// the default code is 12 which maps to unimplemented, override with 0=OK
		HealthyGrpcCodes: jsii.String(
			"0",
		),
		Interval:                awscdk.Duration_Seconds(jsii.Number(5)),
		Timeout:                 awscdk.Duration_Seconds(jsii.Number(2)),
		HealthyThresholdCount:   jsii.Number(3),
		UnhealthyThresholdCount: jsii.Number(3),
	})

	service.Listener().AddTargetGroups(
		jsii.String("GrpcAction"),
		&awselasticloadbalancingv2.AddApplicationTargetGroupsProps{
			TargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{
				targetGroup,
			},
			Conditions: &[]awselasticloadbalancingv2.ListenerCondition{
				awselasticloadbalancingv2.ListenerCondition_PathPatterns(&[]*string{
					jsii.String("/api.poi.v1.*"),
					jsii.String("/api.poi.v1.PoIService/*"),
				}),
			},
			Priority: jsii.Number(10),
		},
	)

	// add TargetGroup for HTTPS REST API on port 8443
	// create http target group for standard https
	restProxyTargetGroup := awselasticloadbalancingv2.NewApplicationTargetGroup(
		stack,
		jsii.String("RestProxytargetGroup"),
		&awselasticloadbalancingv2.ApplicationTargetGroupProps{
			// this is the port of the target container not the port of the ALB!
			Port: jsii.Number(
				restProxyPort,
			),
			Protocol:        awselasticloadbalancingv2.ApplicationProtocol_HTTPS,
			ProtocolVersion: awselasticloadbalancingv2.ApplicationProtocolVersion_HTTP1,
			Vpc:             vpc,
			TargetType:      awselasticloadbalancingv2.TargetType_IP,
			HealthCheck: &awselasticloadbalancingv2.HealthCheck{
				Enabled:                 jsii.Bool(true),
				Path:                    jsii.String("/api/v1/health/liveness"),
				HealthyHttpCodes:        jsii.String("200"),
				Interval:                awscdk.Duration_Seconds(jsii.Number(5)),
				Timeout:                 awscdk.Duration_Seconds(jsii.Number(2)),
				HealthyThresholdCount:   jsii.Number(3),
				UnhealthyThresholdCount: jsii.Number(3),
			},
		},
	)
	service.Listener().
		AddTargetGroups(
			jsii.String("RestProxyTargetGroup"),
			&awselasticloadbalancingv2.AddApplicationTargetGroupsProps{
				TargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{
					restProxyTargetGroup,
				},
				Conditions: &[]awselasticloadbalancingv2.ListenerCondition{
					awselasticloadbalancingv2.ListenerCondition_PathPatterns(&[]*string{
						jsii.String("/api/v1/pois/*"),
					}),
				},
				Priority: jsii.Number(20),
			},
		)
	restProxyTargetGroup.AddTarget(
		service.Service().LoadBalancerTarget(
			&awsecs.LoadBalancerTargetOptions{
				ContainerName: &containerName,
				ContainerPort: jsii.Number(8443),
			},
		),
	)
	// enable communication with ALB on the REST Proxy Port
	service.Service().
		Connections().
		AllowFrom(
			service.LoadBalancer().Connections(),
			awsec2.Port_Tcp(jsii.Number(8443)),
			jsii.String("Allow ALB to reach service"),
		)
	service.LoadBalancer().
		Connections().
		AllowFrom(
			service.Service().Connections(),
			awsec2.Port_Tcp(jsii.Number(8443)),
			jsii.String("Allow servic to reach ALB"),
		)

	return stack
}
