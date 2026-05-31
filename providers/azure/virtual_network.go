// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/chenrui333/terraformer/terraformutils"
)

type VirtualNetworkGenerator struct {
	AzureService
}

func (g *VirtualNetworkGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armnetwork.NewVirtualNetworksClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}

	rg := g.Args["resource_group"].(string)
	var vnets []*armnetwork.VirtualNetwork
	if rg != "" {
		pager := client.NewListPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			vnets = append(vnets, page.Value...)
		}
	} else {
		pager := client.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			vnets = append(vnets, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, vnet := range vnets {
		tferName := terraformutils.TfSanitize(*vnet.Name)
		for _, resource := range resources {
			if tferName == resource.ResourceName {
				*vnet.Name = *vnet.Name + "_" + *vnet.ID
			}
		}

		resources = append(resources, terraformutils.NewSimpleResource(
			*vnet.ID,
			*vnet.Name,
			"azurerm_virtual_network",
			g.ProviderName,
			[]string{}))
	}
	g.Resources = resources
	return nil
}

// NOTE on Virtual Networks and Subnet's:
// Terraform currently provides both a standalone Subnet resource, and allows for Subnets to be defined in-line within the Virtual Network
// resource. At this time you cannot use a Virtual Network with in-line Subnets in conjunction with any Subnet resources.
// Doing so will cause a conflict of Subnet configurations and will overwrite Subnet's.
func (g *VirtualNetworkGenerator) PostConvertHook() error {
	for _, resource := range g.Resources {
		if resource.InstanceInfo.Type != "azurerm_virtual_network" {
			continue
		}
		delete(resource.Item, "subnet")
	}
	return nil
}
