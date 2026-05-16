// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var customerGatewayAllowEmptyValues = []string{"tags."}

type CustomerGatewayGenerator struct {
	AWSService
}

func (g *CustomerGatewayGenerator) createResources(cgws *ec2.DescribeCustomerGatewaysOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, cgw := range cgws.CustomerGateways {
		if cgw.State != nil && (*cgw.State == "deleted" || *cgw.State == "deleting") {
			continue
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(cgw.CustomerGatewayId),
			StringValue(cgw.CustomerGatewayId),
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
	cgws, err := svc.DescribeCustomerGateways(context.TODO(), &ec2.DescribeCustomerGatewaysInput{
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
	g.Resources = g.createResources(cgws)
	return nil
}
