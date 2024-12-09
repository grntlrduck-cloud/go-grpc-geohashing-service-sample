package constructs

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awswafv2"
	awsconstructs "github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type AlbWafACL struct {
	awsconstructs.Construct
	Acl awswafv2.CfnWebACL
}

func NewAlbWafAcl(scope awsconstructs.Construct, id *string) *AlbWafACL {
	myConstruct := awsconstructs.NewConstruct(scope, id)
	acl := awswafv2.NewCfnWebACL(myConstruct, jsii.String("CfnWebACL"), &awswafv2.CfnWebACLProps{
		DefaultAction: &awswafv2.CfnWebACL_DefaultActionProperty{
			Allow: &map[string]interface{}{},
		},
		Scope: jsii.String("REGIONAL"),
		VisibilityConfig: awswafv2.CfnWebACL_VisibilityConfigProperty{
			CloudWatchMetricsEnabled: jsii.Bool(true),
			MetricName:               jsii.Sprintf("wafv2-alb-%s", *id),
			SampledRequestsEnabled:   jsii.Bool(true),
		},
		Description: jsii.String("WAFv2 ACl for ALB"),
		Name:        jsii.Sprintf("wafv2-alb-%s", *id),
		Rules:       ruleSet(),
	})
	return &AlbWafACL{Construct: myConstruct, Acl: acl}
}

func ruleSet() []*awswafv2.CfnWebACL_RuleProperty {
	return []*awswafv2.CfnWebACL_RuleProperty{
		{
			// https://docs.aws.amazon.com/waf/latest/developerguide/aws-managed-rule-groups-baseline.html#aws-managed-rule-groups-baseline-crs
			Name:     jsii.String("AWSManagedRulesCommonRuleSet"),
			Priority: jsii.Number(10),
			Statement: &awswafv2.CfnWebACL_StatementProperty{
				ManagedRuleGroupStatement: &awswafv2.CfnWebACL_ManagedRuleGroupStatementProperty{
					Name:       jsii.String("AWSManagedRulesCommonRuleSet"),
					VendorName: jsii.String("AWS"),
					// to enable testing gRPC from local
					ExcludedRules: &[]*awswafv2.CfnWebACL_ExcludedRuleProperty{
						{
							Name: jsii.String("NoUserAgent_HEADER"),
						},
					},
				},
			},
			VisibilityConfig: &awswafv2.CfnWebACL_VisibilityConfigProperty{
				SampledRequestsEnabled:   jsii.Bool(true),
				CloudWatchMetricsEnabled: jsii.Bool(true),
				MetricName:               jsii.String("AWSManagedRulesCommonRuleSet"),
			},
		},
		{
			// https://docs.aws.amazon.com/waf/latest/developerguide/aws-managed-rule-groups-baseline.html#aws-managed-rule-groups-baseline-crs
			Name:     jsii.String("AWSManagedRulesKnownBadInputsRuleSet"),
			Priority: jsii.Number(20),
			Statement: &awswafv2.CfnWebACL_StatementProperty{
				ManagedRuleGroupStatement: &awswafv2.CfnWebACL_ManagedRuleGroupStatementProperty{
					Name:       jsii.String("AWSManagedRulesKnownBadInputsRuleSet"),
					VendorName: jsii.String("AWS"),
				},
			},
			VisibilityConfig: &awswafv2.CfnWebACL_VisibilityConfigProperty{
				SampledRequestsEnabled:   jsii.Bool(true),
				CloudWatchMetricsEnabled: jsii.Bool(true),
				MetricName:               jsii.String("AWSManagedRulesKnownBadInputsRuleSet"),
			},
		},
		{
			// https://docs.aws.amazon.com/waf/latest/developerguide/aws-managed-rule-groups-baseline.html#aws-managed-rule-groups-baseline-crs
			Name:     jsii.String("AWSManagedRulesAmazonIpReputationList"),
			Priority: jsii.Number(30),
			Statement: &awswafv2.CfnWebACL_StatementProperty{
				ManagedRuleGroupStatement: &awswafv2.CfnWebACL_ManagedRuleGroupStatementProperty{
					Name:       jsii.String("AWSManagedRulesAmazonIpReputationList"),
					VendorName: jsii.String("AWS"),
				},
			},
			VisibilityConfig: &awswafv2.CfnWebACL_VisibilityConfigProperty{
				SampledRequestsEnabled:   jsii.Bool(true),
				CloudWatchMetricsEnabled: jsii.Bool(true),
				MetricName:               jsii.String("AWSManagedRulesAmazonIpReputationList"),
			},
		},
		{
			// Rate Limit the number of requests from one IP
			Name:     jsii.String("RateLimitRequests"),
			Priority: jsii.Number(2),
			Action: awswafv2.CfnWebACL_RuleActionProperty{
				Count: &map[string]interface{}{},
			},
			Statement: &awswafv2.CfnWebACL_StatementProperty{
				RateBasedStatement: &awswafv2.CfnWebACL_RateBasedStatementProperty{
					Limit: jsii.Number(
						1000,
					), // 1000 requests in 5 min = 3.3 requests per second
					AggregateKeyType: jsii.String("IP"),
				},
			},
			VisibilityConfig: &awswafv2.CfnWebACL_VisibilityConfigProperty{
				SampledRequestsEnabled:   jsii.Bool(true),
				CloudWatchMetricsEnabled: jsii.Bool(true),
				MetricName:               jsii.String("RateLimitRequests"),
			},
		},
	}
}
