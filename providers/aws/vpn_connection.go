// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var VpnConnectionAllowEmptyValues = []string{"tags."}

type VpnConnectionGenerator struct {
	AWSService
}

func (g *VpnConnectionGenerator) createResources(vpncs *ec2.DescribeVpnConnectionsOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, vpnc := range vpncs.VpnConnections {
		if vpnc.State == types.VpnStateDeleted || vpnc.State == types.VpnStateDeleting {
			continue
		}
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
	vpncs, err := svc.DescribeVpnConnections(context.TODO(), &ec2.DescribeVpnConnectionsInput{
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
	g.Resources = g.createResources(vpncs)
	return nil
}
