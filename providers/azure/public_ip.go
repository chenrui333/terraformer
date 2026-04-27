// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-03-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type PublicIPGenerator struct {
	AzureService
}

func (g *PublicIPGenerator) listAndAddForPublicIPAddress() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	PublicIPAddressesClient := network.NewPublicIPAddressesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	PublicIPAddressesClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		publicIPAddressIterator network.PublicIPAddressListResultIterator
		err                     error
	)
	if rg := g.Args["resource_group"].(string); rg != "" {
		publicIPAddressIterator, err = PublicIPAddressesClient.ListComplete(ctx, rg)
	} else {
		publicIPAddressIterator, err = PublicIPAddressesClient.ListAllComplete(ctx)
	}
	if err != nil {
		return nil, err
	}
	for publicIPAddressIterator.NotDone() {
		publicIP := publicIPAddressIterator.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*publicIP.ID,
			*publicIP.Name,
			"azurerm_public_ip",
			g.ProviderName,
			[]string{}))

		if err := publicIPAddressIterator.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
	}

	return resources, nil
}

func (g *PublicIPGenerator) listAndAddForPublicIPPrefix() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	PublicIPPrefixesClient := network.NewPublicIPPrefixesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	PublicIPPrefixesClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		publicIPPrefixIterator network.PublicIPPrefixListResultIterator
		err                    error
	)

	if rg := g.Args["resource_group"].(string); rg != "" {
		publicIPPrefixIterator, err = PublicIPPrefixesClient.ListComplete(ctx, rg)
	} else {
		publicIPPrefixIterator, err = PublicIPPrefixesClient.ListAllComplete(ctx)
	}
	if err != nil {
		return nil, err
	}
	for publicIPPrefixIterator.NotDone() {
		publicIPPrefix := publicIPPrefixIterator.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*publicIPPrefix.ID,
			*publicIPPrefix.Name,
			"azurerm_public_ip_prefix",
			g.ProviderName,
			[]string{}))

		if err := publicIPPrefixIterator.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
	}

	return resources, nil
}

func (g *PublicIPGenerator) InitResources() error {
	functions := []func() ([]terraformutils.Resource, error){
		g.listAndAddForPublicIPAddress,
		g.listAndAddForPublicIPPrefix,
	}

	for _, f := range functions {
		resources, err := f()
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	return nil
}
