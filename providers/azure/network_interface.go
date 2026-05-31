// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/chenrui333/terraformer/terraformutils"
)

type NetworkInterfaceGenerator struct {
	AzureService
}

func (g *NetworkInterfaceGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armnetwork.NewInterfacesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}

	rg := g.Args["resource_group"].(string)
	var interfaces []*armnetwork.Interface
	if rg != "" {
		pager := client.NewListPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			interfaces = append(interfaces, page.Value...)
		}
	} else {
		pager := client.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			interfaces = append(interfaces, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, iface := range interfaces {
		resources = append(resources, terraformutils.NewSimpleResource(
			*iface.ID,
			*iface.Name,
			"azurerm_network_interface",
			"azurerm",
			[]string{}))
	}
	g.Resources = resources
	return nil
}
