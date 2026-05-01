// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

type PrivateEndpointGenerator struct {
	AzureService
}

func (az *PrivateEndpointGenerator) listServices() ([]*armnetwork.PrivateLinkService, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewPrivateLinkServicesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armnetwork.PrivateLinkService
	if resourceGroup != "" {
		pager := client.NewListPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			resources = append(resources, page.Value...)
		}
	} else {
		pager := client.NewListBySubscriptionPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			resources = append(resources, page.Value...)
		}
	}
	return resources, nil
}

func (az *PrivateEndpointGenerator) AppendServices(link *armnetwork.PrivateLinkService) {
	az.AppendSimpleResource(*link.ID, *link.Name, "azurerm_private_link_service")
}

func (az *PrivateEndpointGenerator) listEndpoints() ([]*armnetwork.PrivateEndpoint, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewPrivateEndpointsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armnetwork.PrivateEndpoint
	if resourceGroup != "" {
		pager := client.NewListPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			resources = append(resources, page.Value...)
		}
	} else {
		pager := client.NewListBySubscriptionPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			resources = append(resources, page.Value...)
		}
	}
	return resources, nil
}

func (az *PrivateEndpointGenerator) AppendEndpoint(link *armnetwork.PrivateEndpoint) {
	az.AppendSimpleResource(*link.ID, *link.Name, "azurerm_private_endpoint")
}

func (az *PrivateEndpointGenerator) InitResources() error {
	services, err := az.listServices()
	if err != nil {
		return err
	}
	for _, link := range services {
		az.AppendServices(link)
	}
	endpoints, err := az.listEndpoints()
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		az.AppendEndpoint(endpoint)
	}
	return nil
}
