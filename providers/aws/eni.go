// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var EniAllowEmptyValues = []string{"tags."}

type EniGenerator struct {
	AWSService
}

func (EniGenerator) createResources(enis *ec2.DescribeNetworkInterfacesOutput) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, eni := range enis.NetworkInterfaces {
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(eni.NetworkInterfaceId),
			StringValue(eni.NetworkInterfaceId),
			"aws_network_interface",
			"aws",
			EniAllowEmptyValues,
		))
	}
	return resources
}

// Generate TerraformResources from AWS API,
// from each ENI creates 1 TerraformResource.
func (g *EniGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	p := ec2.NewDescribeNetworkInterfacesPaginator(svc, &ec2.DescribeNetworkInterfacesInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
	}
	return nil
}

func (g *EniGenerator) PostConvertHook() error {
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_network_interface" {
			continue
		}
		if _, hasAttachment := r.Item["attachment"]; hasAttachment {
			if attInstance, hasAttachment := r.InstanceState.Attributes["attachment.0.instance"]; !hasAttachment || attInstance == "" {
				delete(r.Item, "attachment")
			}
		}
	}
	return nil
}
