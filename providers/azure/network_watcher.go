// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

type NetworkWatcherGenerator struct {
	AzureService
}

func (az *NetworkWatcherGenerator) listResources() ([]*armnetwork.Watcher, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewWatchersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armnetwork.Watcher
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

func (az *NetworkWatcherGenerator) appendResource(resource *armnetwork.Watcher) {
	az.AppendSimpleResource(*resource.ID, *resource.Name, "azurerm_network_watcher")
}

func (az *NetworkWatcherGenerator) appendFlowLogs(parent *armnetwork.Watcher, resourceGroupID *ResourceID) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewFlowLogsClient(subscriptionID, credential, clientOptions)
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
			az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_network_watcher_flow_log")
		}
	}
	return nil
}

func (az *NetworkWatcherGenerator) appendPacketCaptures(parent *armnetwork.Watcher, resourceGroupID *ResourceID) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewPacketCapturesClient(subscriptionID, credential, clientOptions)
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
			az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_network_packet_capture")
		}
	}
	return nil
}

func (az *NetworkWatcherGenerator) InitResources() error {
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
		err = az.appendFlowLogs(resource, resourceGroupID)
		if err != nil {
			return err
		}
		err = az.appendPacketCaptures(resource, resourceGroupID)
		if err != nil {
			return err
		}
	}
	return nil
}
