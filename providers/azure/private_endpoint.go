// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
)

type PrivateEndpointGenerator struct {
	AzureService
}

func (az *PrivateEndpointGenerator) listServices() ([]network.PrivateLinkService, error) {
	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewPrivateLinkServicesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	var (
		iterator network.PrivateLinkServiceListResultIterator
		err      error
	)
	ctx := context.Background()
	if resourceGroup != "" {
		iterator, err = client.ListComplete(ctx, resourceGroup)
	} else {
		iterator, err = client.ListBySubscriptionComplete(ctx)
	}
	if err != nil {
		return nil, err
	}
	var resources []network.PrivateLinkService
	for iterator.NotDone() {
		item := iterator.Value()
		resources = append(resources, item)
		if err := iterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (az *PrivateEndpointGenerator) AppendServices(link *network.PrivateLinkService) {
	az.AppendSimpleResource(*link.ID, *link.Name, "azurerm_private_link_service")
}

func (az *PrivateEndpointGenerator) listEndpoints() ([]network.PrivateEndpoint, error) {
	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewPrivateEndpointsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	var (
		iterator network.PrivateEndpointListResultIterator
		err      error
	)
	ctx := context.Background()
	if resourceGroup != "" {
		iterator, err = client.ListComplete(ctx, resourceGroup)
	} else {
		iterator, err = client.ListBySubscriptionComplete(ctx)
	}
	if err != nil {
		return nil, err
	}
	var resources []network.PrivateEndpoint
	for iterator.NotDone() {
		item := iterator.Value()
		resources = append(resources, item)
		if err := iterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (az *PrivateEndpointGenerator) AppendEndpoint(link *network.PrivateEndpoint) {
	az.AppendSimpleResource(*link.ID, *link.Name, "azurerm_private_endpoint")
}

func (az *PrivateEndpointGenerator) InitResources() error {
	services, err := az.listServices()
	if err != nil {
		return err
	}
	for _, link := range services {
		az.AppendServices(&link)
	}
	endpoints, err := az.listEndpoints()
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		az.AppendEndpoint(&endpoint)
	}
	return nil
}
