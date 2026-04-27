// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type VirtualNetworkGenerator struct {
	AzureService
}

func (g VirtualNetworkGenerator) createResources(ctx context.Context, iterator network.VirtualNetworkListResultIterator) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for iterator.NotDone() {
		virtualNetwork := iterator.Value()
		tferName := terraformutils.TfSanitize(*virtualNetwork.Name)
		for _, resource := range resources {
			if tferName == resource.ResourceName {
				*virtualNetwork.Name = *virtualNetwork.Name + "_" + *virtualNetwork.ID
			}
		}

		resources = append(resources, terraformutils.NewSimpleResource(
			*virtualNetwork.ID,
			*virtualNetwork.Name,
			"azurerm_virtual_network",
			g.ProviderName,
			[]string{}))
		if err := iterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (g *VirtualNetworkGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	virtualNetworkClient := network.NewVirtualNetworksClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

	virtualNetworkClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		output network.VirtualNetworkListResultIterator
		err    error
	)

	if rg := g.Args["resource_group"].(string); rg != "" {
		output, err = virtualNetworkClient.ListComplete(ctx, rg)
	} else {
		output, err = virtualNetworkClient.ListAllComplete(ctx)
	}
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(ctx, output)
	return err
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
