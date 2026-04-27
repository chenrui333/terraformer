// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/resourcegroups"
	"github.com/chenrui333/terraformer/terraformutils"
)

var resourcegroupsAllowEmptyValues = []string{"tags."}

type ResourceGroupsGenerator struct {
	AWSService
}

func (g *ResourceGroupsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := resourcegroups.NewFromConfig(config)
	p := resourcegroups.NewListGroupsPaginator(svc, &resourcegroups.ListGroupsInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, groupIdentifier := range page.GroupIdentifiers {
			groupName := StringValue(groupIdentifier.GroupName)
			resources = append(resources, terraformutils.NewSimpleResource(
				groupName,
				groupName,
				"aws_resourcegroups_group",
				"aws",
				resourcegroupsAllowEmptyValues))
		}
	}
	g.Resources = resources
	return nil
}
