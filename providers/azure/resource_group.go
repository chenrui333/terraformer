// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ResourceGroupGenerator struct {
	AzureService
}

func (g ResourceGroupGenerator) createResources(groupListResultIterator resources.GroupListResultIterator) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for groupListResultIterator.NotDone() {
		group := groupListResultIterator.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*group.ID,
			*group.Name,
			"azurerm_resource_group",
			"azurerm",
			[]string{}))
		if err := groupListResultIterator.Next(); err != nil {
			log.Println(err)
			break
		}
	}
	return resources
}

func (g *ResourceGroupGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	groupsClient := resources.NewGroupsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

	groupsClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	if rg := g.Args["resource_group"].(string); rg != "" {
		group, err := groupsClient.Get(ctx, rg)
		if err != nil {
			return err
		}
		g.Resources = []terraformutils.Resource{
			terraformutils.NewSimpleResource(
				*group.ID,
				*group.Name,
				"azurerm_resource_group",
				"azurerm",
				[]string{}),
		}
		return nil
	}
	output, err := groupsClient.ListComplete(ctx, "", nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
