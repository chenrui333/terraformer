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

		vpceID := StringValue(vpcEndpoint.VpcEndpointId)
		resources = append(resources, terraformutils.NewSimpleResource(
			vpceID,
			vpceID,
			"aws_vpc_endpoint",
			"aws",
			VpcEndpointAllowEmptyValues,
		))

		for _, rtID := range vpcEndpoint.RouteTableIds {
			id := vpceID + "/" + rtID
			resources = append(resources, terraformutils.NewResource(
				id,
				id,
				"aws_vpc_endpoint_route_table_association",
				"aws",
				map[string]string{
					"vpc_endpoint_id": vpceID,
					"route_table_id":  rtID,
				},
				VpcEndpointAllowEmptyValues,
				map[string]interface{}{},
			))
		}

		for _, subnetID := range vpcEndpoint.SubnetIds {
			id := vpceID + "/" + subnetID
			resources = append(resources, terraformutils.NewResource(
				id,
				id,
				"aws_vpc_endpoint_subnet_association",
				"aws",
				map[string]string{
					"vpc_endpoint_id": vpceID,
					"subnet_id":       subnetID,
				},
				VpcEndpointAllowEmptyValues,
				map[string]interface{}{},
			))
		}
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
