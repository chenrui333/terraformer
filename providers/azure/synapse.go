// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/synapse/armsynapse"
)

type SynapseGenerator struct {
	AzureService
}

func (az *SynapseGenerator) listWorkspaces() ([]*armsynapse.Workspace, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armsynapse.NewWorkspacesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armsynapse.Workspace
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

func (az *SynapseGenerator) appendWorkspace(workspace *armsynapse.Workspace) {
	az.AppendSimpleResource(*workspace.ID, *workspace.Name, "azurerm_synapse_workspace")
}

func (az *SynapseGenerator) appendSQLPools(workspace *armsynapse.Workspace, workspaceRg *ResourceID) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armsynapse.NewSQLPoolsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}
	ctx := context.Background()
	pager := client.NewListByWorkspacePager(workspaceRg.ResourceGroup, *workspace.Name, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range page.Value {
			az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_synapse_sql_pool")
		}
	}
	return nil
}

func (az *SynapseGenerator) appendSparkPools(workspace *armsynapse.Workspace, workspaceRg *ResourceID) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armsynapse.NewBigDataPoolsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}
	ctx := context.Background()
	pager := client.NewListByWorkspacePager(workspaceRg.ResourceGroup, *workspace.Name, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range page.Value {
			az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_synapse_spark_pool")
		}
	}
	return nil
}

func (az *SynapseGenerator) appendFirewallRule(workspace *armsynapse.Workspace, workspaceRg *ResourceID) error {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armsynapse.NewIPFirewallRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}
	ctx := context.Background()
	pager := client.NewListByWorkspacePager(workspaceRg.ResourceGroup, *workspace.Name, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range page.Value {
			az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_synapse_firewall_rule")
		}
	}
	return nil
}

func (az *SynapseGenerator) listPrivateLinkHubs() ([]*armsynapse.PrivateLinkHub, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armsynapse.NewPrivateLinkHubsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armsynapse.PrivateLinkHub
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

func (az *SynapseGenerator) appendPrivateLinkHub(hub *armsynapse.PrivateLinkHub) {
	az.AppendSimpleResource(*hub.ID, *hub.Name, "azurerm_synapse_private_link_hub")
}

func (az *SynapseGenerator) InitResources() error {
	workspaces, err := az.listWorkspaces()
	if err != nil {
		return err
	}
	for _, workspace := range workspaces {
		az.appendWorkspace(workspace)
		workspaceRg, err := ParseAzureResourceID(*workspace.ID)
		if err != nil {
			return err
		}
		err = az.appendSQLPools(workspace, workspaceRg)
		if err != nil {
			return err
		}
		err = az.appendSparkPools(workspace, workspaceRg)
		if err != nil {
			return err
		}
		err = az.appendFirewallRule(workspace, workspaceRg)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	hubs, err := az.listPrivateLinkHubs()
	if err == nil {
		for _, hub := range hubs {
			az.appendPrivateLinkHub(hub)
		}
	}
	return nil
}
