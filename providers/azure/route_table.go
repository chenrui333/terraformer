// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

type RouteTableGenerator struct {
	AzureService
}

func (az *RouteTableGenerator) listResources() ([]*armnetwork.RouteTable, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewRouteTablesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armnetwork.RouteTable
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
		pager := client.NewListAllPager(nil)
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

func (az *RouteTableGenerator) appendResource(resource *armnetwork.RouteTable) {
	az.AppendSimpleResourceWithDuplicateCheck(*resource.ID, *resource.Name, "azurerm_route_table")
}

func (az *RouteTableGenerator) appendRoutes(parent *armnetwork.RouteTable, resourceGroupID *ResourceID) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewRoutesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}
	ctx := context.Background()
	pager := client.NewListPager(resourceGroupID.ResourceGroup, *parent.Name, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range page.Value {
			az.AppendSimpleResourceWithDuplicateCheck(*item.ID, *item.Name, "azurerm_route")
		}
	}
	return nil
}

func (az *RouteTableGenerator) listRouteFilters() ([]*armnetwork.RouteFilter, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewRouteFiltersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armnetwork.RouteFilter
	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			resources = append(resources, page.Value...)
		}
	} else {
		pager := client.NewListPager(nil)
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

func (az *RouteTableGenerator) appendRouteFilters(resource *armnetwork.RouteFilter) {
	az.AppendSimpleResource(*resource.ID, *resource.Name, "azurerm_route_filter")
}

func (az *RouteTableGenerator) InitResources() error {
	resources, err := az.listResources()
	if err != nil {
		return err
	}
	for _, resource := range resources {
		az.appendResource(resource)
		resourceGroupID, err := ParseAzureResourceID(*resource.ID)
		if err != nil {
			return err
		}
		err = az.appendRoutes(resource, resourceGroupID)
		if err != nil {
			return err
		}
	}

	filters, err := az.listRouteFilters()
	if err != nil {
		return err
	}
	for _, resource := range filters {
		az.appendRouteFilters(resource)
	}
	return nil
}
