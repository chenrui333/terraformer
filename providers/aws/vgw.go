// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var VpnAllowEmptyValues = []string{"tags."}

type VpnGatewayGenerator struct {
	AWSService
}

func (VpnGatewayGenerator) createResources(vpnGws *ec2.DescribeVpnGatewaysOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, vpnGw := range vpnGws.VpnGateways {
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(vpnGw.VpnGatewayId),
			StringValue(vpnGw.VpnGatewayId),
			"aws_vpn_gateway",
			"aws",
			VpnAllowEmptyValues,
		))
	}
	return resources
}

// Generate TerraformResources from AWS API,
// from each vpn gateway create 1 TerraformResource.
// Need VpnGatewayId as ID for terraform resource
func (g *VpnGatewayGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	vpnGws, err := svc.DescribeVpnGateways(context.TODO(), &ec2.DescribeVpnGatewaysInput{})
	if err != nil {
		return err
	}
	g.Resources = g.createResources(vpnGws)
	return nil
}
