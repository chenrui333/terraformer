// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var customerGatewayAllowEmptyValues = []string{"tags."}

type CustomerGatewayGenerator struct {
	AWSService
}

func (CustomerGatewayGenerator) createResources(cgws *ec2.DescribeCustomerGatewaysOutput) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, cgws := range cgws.CustomerGateways {
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(cgws.CustomerGatewayId),
			StringValue(cgws.CustomerGatewayId),
			"aws_customer_gateway",
			"aws",
			customerGatewayAllowEmptyValues,
		))
	}
	return resources
}

// Generate TerraformResources from AWS API,
// from each customer gateway create 1 TerraformResource.
// Need CustomerGatewayId as ID for terraform resource
func (g *CustomerGatewayGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	cgws, err := svc.DescribeCustomerGateways(context.TODO(), &ec2.DescribeCustomerGatewaysInput{})
	if err != nil {
		return err
	}
	g.Resources = g.createResources(cgws)
	return nil
}
