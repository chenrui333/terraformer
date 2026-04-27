// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var peeringAllowEmptyValues = []string{"tags."}

type VpcPeeringConnectionGenerator struct {
	AWSService
}

func (g *VpcPeeringConnectionGenerator) createResources(peerings *ec2.DescribeVpcPeeringConnectionsOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, peering := range peerings.VpcPeeringConnections {
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(peering.VpcPeeringConnectionId),
			StringValue(peering.VpcPeeringConnectionId),
			"aws_vpc_peering_connection",
			"aws",
			peeringAllowEmptyValues,
		))
	}

	return resources
}

// Generate TerraformResources from AWS API,
// create terraform resource for each VPC Peering Connection
func (g *VpcPeeringConnectionGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	p := ec2.NewDescribeVpcPeeringConnectionsPaginator(svc, &ec2.DescribeVpcPeeringConnectionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
	}
	return nil
}
