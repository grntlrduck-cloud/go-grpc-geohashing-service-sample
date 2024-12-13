package constructs

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// The function gets the default mini VPC with only one AZ per subnet
func LandingZoneVPC(scope constructs.Construct, id string) awsec2.IVpc {
	vpc := awsec2.Vpc_FromVpcAttributes(scope, jsii.String(id), &awsec2.VpcAttributes{
		VpcId:             awscdk.Fn_ImportValue(jsii.String("LandingZoneVpcId")),
		AvailabilityZones: jsii.Strings("eu-west-1a", "eu-west-1b"),
		IsolatedSubnetIds: &[]*string{
			awscdk.Fn_ImportValue(jsii.String("LandingZoneIsolatedSubnet1Id")),
			awscdk.Fn_ImportValue(jsii.String("LandingZoneIsolatedSubnet2Id")),
		},
		PublicSubnetIds: &[]*string{
			awscdk.Fn_ImportValue(jsii.String("LandingZonePublicSubnet1Id")),
			awscdk.Fn_ImportValue(jsii.String("LandingZonePublicSubnet2Id")),
		},
		PrivateSubnetIds: &[]*string{
			awscdk.Fn_ImportValue(jsii.String("LandingZonePrivateSubnet1Id")),
			awscdk.Fn_ImportValue(jsii.String("LandingZonePrivateSubnet2Id")),
		},
		// only the route table for private subnet are interesting
		PrivateSubnetRouteTableIds: &[]*string{
			awscdk.Fn_ImportValue(
				jsii.String("LandingZonePrivateSubnet1RouteTableId"),
			),
			awscdk.Fn_ImportValue(
				jsii.String("LandingZonePrivateSubnet2RouteTableId"),
			),
		},
	})
	awscdk.Annotations_Of(vpc).AcknowledgeWarning(
		jsii.String("@aws-cdk/aws-ec2:noSubnetRouteTableId"),
		jsii.String("This warning can be ignored because in most cases you are not interested in the other RouteTables"),
	)
	return vpc
}

// This function gets the HostedZone from attributes for certificate and record creation
func LandingHostedZone(scope constructs.Construct, id string) awsroute53.IHostedZone {
	return awsroute53.HostedZone_FromHostedZoneAttributes(
		scope,
		jsii.String(id),
		&awsroute53.HostedZoneAttributes{
			HostedZoneId: awscdk.Fn_ImportValue(jsii.String("LandingZoneHostedZoneId")),
			ZoneName:     awscdk.Fn_ImportValue(jsii.String("LandingZoneHostedZoneName")),
		},
	)
}
