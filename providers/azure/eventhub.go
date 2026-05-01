// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub"
)

type EventHubGenerator struct {
	AzureService
}

func (az *EventHubGenerator) listNamespaces() ([]*armeventhub.EHNamespace, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armeventhub.NewNamespacesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armeventhub.EHNamespace
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

func (az *EventHubGenerator) AppendNamespace(namespace *armeventhub.EHNamespace) {
	az.AppendSimpleResource(*namespace.ID, *namespace.Name, "azurerm_eventhub_namespace")
}

func (az *EventHubGenerator) appendEventHubs(namespace *armeventhub.EHNamespace, namespaceRg *ResourceID) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armeventhub.NewEventHubsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}
	ctx := context.Background()
	pager := client.NewListByNamespacePager(namespaceRg.ResourceGroup, *namespace.Name, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range page.Value {
			az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_eventhub")
			if err := az.appendConsumerGroups(namespace, namespaceRg, *item.Name); err != nil {
				log.Println(err)
				return err
			}
		}
	}
	return nil
}

func (az *EventHubGenerator) appendConsumerGroups(namespace *armeventhub.EHNamespace, namespaceRg *ResourceID, eventHubName string) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armeventhub.NewConsumerGroupsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}
	ctx := context.Background()
	pager := client.NewListByEventHubPager(namespaceRg.ResourceGroup, *namespace.Name, eventHubName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range page.Value {
			az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_eventhub_consumer_group")
		}
	}
	return nil
}

func (az *EventHubGenerator) appendAuthorizationRules(namespace *armeventhub.EHNamespace, namespaceRg *ResourceID) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armeventhub.NewNamespacesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}
	ctx := context.Background()
	pager := client.NewListAuthorizationRulesPager(namespaceRg.ResourceGroup, *namespace.Name, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range page.Value {
			az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_eventhub_namespace_authorization_rule")
		}
	}
	return nil
}

func (az *EventHubGenerator) InitResources() error {
	namespaces, err := az.listNamespaces()
	if err != nil {
		return err
	}
	for _, namespace := range namespaces {
		az.AppendNamespace(namespace)
		namespaceRg, err := ParseAzureResourceID(*namespace.ID)
		if err != nil {
			return err
		}
		err = az.appendEventHubs(namespace, namespaceRg)
		if err != nil {
			return err
		}
		err = az.appendAuthorizationRules(namespace, namespaceRg)
		if err != nil {
			return err
		}
	}
	return nil
}
