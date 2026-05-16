// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var VpcEndpointAllowEmptyValues = []string{"tags."}

type VpcEndpointGenerator struct {
	AWSService
}

func (g *VpcEndpointGenerator) createResources(vpceps *ec2.DescribeVpcEndpointsOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, vpcEndpoint := range vpceps.VpcEndpoints {
		if vpcEndpoint.State == types.StateDeleted || vpcEndpoint.State == types.StateDeleting {
			continue
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(vpcEndpoint.VpcEndpointId),
			StringValue(vpcEndpoint.VpcEndpointId),
			"aws_vpc_endpoint",
			"aws",
			VpcEndpointAllowEmptyValues,
		))
	}
	return resources
}

// Generate TerraformResources from AWS API,
// from each vpc endpoint create 1 TerraformResource.
// Need VpcEndpointId as ID for terraform resource
func (g *VpcEndpointGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	p := ec2.NewDescribeVpcEndpointsPaginator(svc, &ec2.DescribeVpcEndpointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
	}
	return nil
}
