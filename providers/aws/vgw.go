// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var VpnAllowEmptyValues = []string{"tags."}

type VpnGatewayGenerator struct {
	AWSService
}

func (g *VpnGatewayGenerator) createResources(vpnGws *ec2.DescribeVpnGatewaysOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, vpnGw := range vpnGws.VpnGateways {
		if vpnGw.State == types.VpnStateDeleted || vpnGw.State == types.VpnStateDeleting {
			continue
		}
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
	vpnGws, err := svc.DescribeVpnGateways(context.TODO(), &ec2.DescribeVpnGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"pending", "available"},
			},
		},
	})
	if err != nil {
		return err
	}
	g.Resources = g.createResources(vpnGws)
	return nil
}
