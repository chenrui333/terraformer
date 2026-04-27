// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-08-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type NetworkInterfaceGenerator struct {
	AzureService
}

func (g NetworkInterfaceGenerator) createResources(interfaceListResult network.InterfaceListResultIterator) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for interfaceListResult.NotDone() {
		networkInterface := interfaceListResult.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*networkInterface.ID,
			*networkInterface.Name,
			"azurerm_network_interface",
			"azurerm",
			[]string{}))
		if err := interfaceListResult.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (g *NetworkInterfaceGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	interfacesClient := network.NewInterfacesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

	interfacesClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)
	var (
		output network.InterfaceListResultIterator
		err    error
	)
	if rg := g.Args["resource_group"].(string); rg != "" {
		output, err = interfacesClient.ListComplete(ctx, rg)
	} else {
		output, err = interfacesClient.ListAllComplete(ctx)
	}
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(output)
	return err
}
