// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ContainerGenerator struct {
	AzureService
}

func (g *ContainerGenerator) listAndAddForContainerGroup() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, resourceGroup, credential, clientOptions := g.getClientArgs()

	client, err := armcontainerinstance.NewContainerGroupsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var groups []*armcontainerinstance.ContainerGroup
	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			groups = append(groups, page.Value...)
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			groups = append(groups, page.Value...)
		}
	}

	for _, containerGroup := range groups {
		resources = append(resources, terraformutils.NewSimpleResource(
			*containerGroup.ID,
			*containerGroup.Name,
			"azurerm_container_group",
			g.ProviderName,
			[]string{}))
	}

	return resources, nil
}

func (g *ContainerGenerator) listRegistryWebhooks(resourceGroupName string, registryName string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armcontainerregistry.NewWebhooksClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(resourceGroupName, registryName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, webhook := range page.Value {
			resources = append(resources, terraformutils.NewSimpleResource(
				*webhook.ID,
				*webhook.Name,
				"azurerm_container_registry_webhook",
				g.ProviderName,
				[]string{}))
		}
	}
	return resources, nil
}

func (g *ContainerGenerator) listAndAddForContainerRegistry() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, resourceGroup, credential, clientOptions := g.getClientArgs()

	client, err := armcontainerregistry.NewRegistriesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var registries []*armcontainerregistry.Registry
	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			registries = append(registries, page.Value...)
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			registries = append(registries, page.Value...)
		}
	}

	for _, containerRegistry := range registries {
		resources = append(resources, terraformutils.NewSimpleResource(
			*containerRegistry.ID,
			*containerRegistry.Name,
			"azurerm_container_registry",
			g.ProviderName,
			[]string{}))

		id, err := ParseAzureResourceID(*containerRegistry.ID)
		if err != nil {
			return nil, err
		}

		webhooks, err := g.listRegistryWebhooks(id.ResourceGroup, *containerRegistry.Name)
		if err != nil {
			return resources, err
		}
		resources = append(resources, webhooks...)
	}

	return resources, nil
}

func (g *ContainerGenerator) InitResources() error {
	functions := []func() ([]terraformutils.Resource, error){
		g.listAndAddForContainerGroup,
		g.listAndAddForContainerRegistry,
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
