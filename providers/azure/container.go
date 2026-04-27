// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/containerinstance/mgmt/2018-10-01/containerinstance"
	"github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-05-01/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ContainerGenerator struct {
	AzureService
}

func (g *ContainerGenerator) listAndAddForContainerGroup() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	ContainerGroupsClient := containerinstance.NewContainerGroupsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	ContainerGroupsClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		containerGroupIterator containerinstance.ContainerGroupListResultIterator
		err                    error
	)

	if rg := g.Args["resource_group"].(string); rg != "" {
		containerGroupIterator, err = ContainerGroupsClient.ListByResourceGroupComplete(ctx, rg)
	} else {
		containerGroupIterator, err = ContainerGroupsClient.ListComplete(ctx)
	}
	if err != nil {
		return nil, err
	}
	for containerGroupIterator.NotDone() {
		containerGroup := containerGroupIterator.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*containerGroup.ID,
			*containerGroup.Name,
			"azurerm_container_group",
			g.ProviderName,
			[]string{}))

		if err := containerGroupIterator.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
	}

	return resources, nil
}

func (g *ContainerGenerator) listRegistryWebhooks(resourceGroupName string, registryName string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	WebhooksClient := containerregistry.NewWebhooksClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	WebhooksClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	webhookIterator, err := WebhooksClient.ListComplete(ctx, resourceGroupName, registryName)
	if err != nil {
		return nil, err
	}
	for webhookIterator.NotDone() {
		webhook := webhookIterator.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*webhook.ID,
			*webhook.Name,
			"azurerm_container_registry_webhook",
			g.ProviderName,
			[]string{}))
		if err := webhookIterator.Next(); err != nil {
			log.Println(err)
			break
		}
	}
	return resources, nil
}

func (g *ContainerGenerator) listAndAddForContainerRegistry() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	ContainerRegistriesClient := containerregistry.NewRegistriesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	ContainerRegistriesClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		containerRegistryIterator containerregistry.RegistryListResultIterator
		err                       error
	)

	if rg := g.Args["resource_group"].(string); rg != "" {
		containerRegistryIterator, err = ContainerRegistriesClient.ListByResourceGroupComplete(ctx, rg)
	} else {
		containerRegistryIterator, err = ContainerRegistriesClient.ListComplete(ctx)
	}
	if err != nil {
		return nil, err
	}
	for containerRegistryIterator.NotDone() {
		containerRegistry := containerRegistryIterator.Value()
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
			return nil, err
		}
		resources = append(resources, webhooks...)

		if err := containerRegistryIterator.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
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
