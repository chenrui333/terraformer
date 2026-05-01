// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/chenrui333/terraformer/terraformutils"
)

type PublicIPGenerator struct {
	AzureService
}

func (g *PublicIPGenerator) listAndAddForPublicIPAddress() ([]terraformutils.Resource, error) {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	rg := g.Args["resource_group"].(string)
	var publicIPs []*armnetwork.PublicIPAddress
	if rg != "" {
		pager := client.NewListPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			publicIPs = append(publicIPs, page.Value...)
		}
	} else {
		pager := client.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			publicIPs = append(publicIPs, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, ip := range publicIPs {
		resources = append(resources, terraformutils.NewSimpleResource(
			*ip.ID,
			*ip.Name,
			"azurerm_public_ip",
			g.ProviderName,
			[]string{}))
	}

	return resources, nil
}

func (g *PublicIPGenerator) listAndAddForPublicIPPrefix() ([]terraformutils.Resource, error) {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armnetwork.NewPublicIPPrefixesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	rg := g.Args["resource_group"].(string)
	var prefixes []*armnetwork.PublicIPPrefix
	if rg != "" {
		pager := client.NewListPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			prefixes = append(prefixes, page.Value...)
		}
	} else {
		pager := client.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			prefixes = append(prefixes, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, prefix := range prefixes {
		resources = append(resources, terraformutils.NewSimpleResource(
			*prefix.ID,
			*prefix.Name,
			"azurerm_public_ip_prefix",
			g.ProviderName,
			[]string{}))
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
