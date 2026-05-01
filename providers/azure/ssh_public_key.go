// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
)

type SSHPublicKeyGenerator struct {
	AzureService
}

func (az *SSHPublicKeyGenerator) listResources() ([]*armcompute.SSHPublicKeyResource, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armcompute.NewSSHPublicKeysClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armcompute.SSHPublicKeyResource
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

func (az *SSHPublicKeyGenerator) appendResource(resource *armcompute.SSHPublicKeyResource) {
	az.AppendSimpleResource(*resource.ID, *resource.Name, "azurerm_ssh_public_key")
}

func (az *SSHPublicKeyGenerator) InitResources() error {
	resources, err := az.listResources()
	if err != nil {
		return err
	}
	for _, resource := range resources {
		az.appendResource(resource)
	}
	return nil
}
