// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var VpcEndpointAllowEmptyValues = []string{"tags."}

type VpcEndpointGenerator struct {
	AWSService
}

func (g *VpcEndpointGenerator) createResources(vpceps *ec2.DescribeVpcEndpointsOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, vpcEndpoint := range vpceps.VpcEndpoints {
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(vpcEndpoint.VpcEndpointId),
			StringValue(vpcEndpoint.VpcEndpointId),
			"aws_vpc_endpoint",
			"aws",
			VpcAllowEmptyValues,
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
	vpceps, err := svc.DescribeVpcEndpoints(context.TODO(), &ec2.DescribeVpcEndpointsInput{})
	if err != nil {
		return err
	}
	g.Resources = g.createResources(vpceps)
	return nil
}
