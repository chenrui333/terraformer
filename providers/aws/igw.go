// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var IgwAllowEmptyValues = []string{"tags."}

type IgwGenerator struct {
	AWSService
}

// Generate TerraformResources from AWS API,
// from each Internet gateway create 1 TerraformResource.
// Need InternetGatewayId as ID for terraform resource
func (g *IgwGenerator) createResources(igws *ec2.DescribeInternetGatewaysOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, internetGateway := range igws.InternetGateways {
		if len(internetGateway.Attachments) == 0 {
			continue
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(internetGateway.InternetGatewayId),
			StringValue(internetGateway.InternetGatewayId),
			"aws_internet_gateway",
			"aws",
			IgwAllowEmptyValues,
		))
	}
	return resources
}

func (g *IgwGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	p := ec2.NewDescribeInternetGatewaysPaginator(svc, &ec2.DescribeInternetGatewaysInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
	}
	return nil
}
