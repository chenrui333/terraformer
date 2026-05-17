// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var ngwAllowEmptyValues = []string{"tags."}

type NatGatewayGenerator struct {
	AWSService
}

func (g *NatGatewayGenerator) createResources(ngws *ec2.DescribeNatGatewaysOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, ngw := range ngws.NatGateways {
		if ngw.State == types.NatGatewayStateDeleted || ngw.State == types.NatGatewayStateDeleting {
			continue
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(ngw.NatGatewayId),
			StringValue(ngw.NatGatewayId),
			"aws_nat_gateway",
			"aws",
			ngwAllowEmptyValues,
		))
	}

	return resources
}

// Generate TerraformResources from AWS API,
// create terraform resource for each NAT Gateways
func (g *NatGatewayGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	p := ec2.NewDescribeNatGatewaysPaginator(svc, &ec2.DescribeNatGatewaysInput{
		Filter: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"available", "pending", "failed"},
			},
		},
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
	}
	return nil
}
