// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var VpnConnectionAllowEmptyValues = []string{"tags."}

type VpnConnectionGenerator struct {
	AWSService
}

func (VpnConnectionGenerator) createResources(vpncs *ec2.DescribeVpnConnectionsOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, vpnc := range vpncs.VpnConnections {
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(vpnc.VpnConnectionId),
			StringValue(vpnc.VpnConnectionId),
			"aws_vpn_connection",
			"aws",
			VpnConnectionAllowEmptyValues,
		))
	}
	return resources
}

// Generate TerraformResources from AWS API,
// from each vpn connection create 1 TerraformResource.
// Need VpnConnectionId as ID for terraform resource
func (g *VpnConnectionGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	vpncs, err := svc.DescribeVpnConnections(context.TODO(), &ec2.DescribeVpnConnectionsInput{})
	if err != nil {
		return err
	}
	g.Resources = g.createResources(vpncs)
	return nil
}
