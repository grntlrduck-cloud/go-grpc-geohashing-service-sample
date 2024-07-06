package utils

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

func OverrideLogicalId(construct constructs.Node, idOverride string) {
	var resource awscdk.CfnResource
	jsii.Get(construct, "defaultChild", &resource) // nolint:staticcheck
  resource.OverrideLogicalId(jsii.String(idOverride))
}
