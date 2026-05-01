// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ResourceGroupGenerator struct {
	AzureService
}

func (g *ResourceGroupGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armresources.NewResourceGroupsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}

	if rg := g.Args["resource_group"].(string); rg != "" {
		group, err := client.Get(ctx, rg, nil)
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

	var resources []terraformutils.Resource
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, group := range page.Value {
			resources = append(resources, terraformutils.NewSimpleResource(
				*group.ID,
				*group.Name,
				"azurerm_resource_group",
				"azurerm",
				[]string{}))
		}
	}
	g.Resources = resources
	return nil
}
