// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/chenrui333/terraformer/terraformutils"
)

var NaclAllowEmptyValues = []string{"tags."}

type NaclGenerator struct {
	AWSService
}

func (NaclGenerator) createResources(nacls *ec2.DescribeNetworkAclsOutput) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	var resourceType string
	for _, nacl := range nacls.NetworkAcls {
		if nacl.IsDefault != nil && *nacl.IsDefault {
			resourceType = "aws_default_network_acl"
		} else {
			resourceType = "aws_network_acl"
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(nacl.NetworkAclId),
			StringValue(nacl.NetworkAclId),
			resourceType,
			"aws",
			NaclAllowEmptyValues))
	}
	return resources
}

// Generate TerraformResources from AWS API,
// from each network ACL create 1 TerraformResource.
// Need NetworkAclId as ID for terraform resource
func (g *NaclGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	p := ec2.NewDescribeNetworkAclsPaginator(svc, &ec2.DescribeNetworkAclsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
	}
	return nil
}
