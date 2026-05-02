// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var eipAllowEmptyValues = []string{"tags."}

type ElasticIPGenerator struct {
	AWSService
}

func (g *ElasticIPGenerator) createElasticIpsResources(svc *ec2.Client) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	addresses, err := svc.DescribeAddresses(context.TODO(), &ec2.DescribeAddressesInput{})

	if err != nil {
		return nil, fmt.Errorf("describe elastic IP addresses: %w", err)
	}

	for _, eip := range addresses.Addresses {
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(eip.AllocationId),
			StringValue(eip.AllocationId),
			"aws_eip",
			"aws",
			eipAllowEmptyValues,
		))
	}

	return resources, nil
}

// Generate TerraformResources from AWS API,
// create terraform resource for each elastic IPs
func (g *ElasticIPGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)

	resources, err := g.createElasticIpsResources(svc)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
