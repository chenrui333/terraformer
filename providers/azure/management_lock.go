// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armlocks"
)

type ManagementLockGenerator struct {
	AzureService
}

func (az *ManagementLockGenerator) listResources() ([]*armlocks.ManagementLockObject, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armlocks.NewManagementLocksClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armlocks.ManagementLockObject
	if resourceGroup != "" {
		pager := client.NewListAtResourceGroupLevelPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			resources = append(resources, page.Value...)
		}
	} else {
		pager := client.NewListAtSubscriptionLevelPager(nil)
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

func (az *ManagementLockGenerator) appendResource(resource *armlocks.ManagementLockObject) {
	az.AppendSimpleResource(*resource.ID, *resource.Name, "azurerm_management_lock")
}

func (az *ManagementLockGenerator) InitResources() error {
	resources, err := az.listResources()
	if err != nil {
		return err
	}
	for _, resource := range resources {
		az.appendResource(resource)
	}
	return nil
}
