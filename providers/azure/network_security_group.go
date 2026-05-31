// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

type NetworkSecurityGroupGenerator struct {
	AzureService
}

func (az *NetworkSecurityGroupGenerator) listResources() ([]*armnetwork.SecurityGroup, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewSecurityGroupsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armnetwork.SecurityGroup
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

func (az *NetworkSecurityGroupGenerator) appendResource(resource *armnetwork.SecurityGroup) {
	az.AppendSimpleResourceWithDuplicateCheck(*resource.ID, *resource.Name, "azurerm_network_security_group")
}

func (az *NetworkSecurityGroupGenerator) appendRules(parent *armnetwork.SecurityGroup, resourceGroupID *ResourceID) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewSecurityRulesClient(subscriptionID, credential, clientOptions)
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
			az.AppendSimpleResourceWithDuplicateCheck(*item.ID, *item.Name, "azurerm_network_security_rule")
		}
	}
	return nil
}

func (az *NetworkSecurityGroupGenerator) InitResources() error {
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
		err = az.appendRules(resource, resourceGroupID)
		if err != nil {
			return err
		}
	}
	return nil
}
