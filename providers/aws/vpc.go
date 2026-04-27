// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var VpcAllowEmptyValues = []string{"tags."}

type VpcGenerator struct {
	AWSService
}

func (VpcGenerator) createResources(vpcs *ec2.DescribeVpcsOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, vpc := range vpcs.Vpcs {
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(vpc.VpcId),
			StringValue(vpc.VpcId),
			"aws_vpc",
			"aws",
			VpcAllowEmptyValues,
		))
	}
	return resources
}

// Generate TerraformResources from AWS API,
// from each vpc create 1 TerraformResource.
// Need VpcId as ID for terraform resource
func (g *VpcGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	p := ec2.NewDescribeVpcsPaginator(svc, &ec2.DescribeVpcsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
	}
	return nil
}
