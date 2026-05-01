// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
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

func (az *SynapseGenerator) appendManagedPrivateEndpoint(workspace *armsynapse.Workspace) error {
	if workspace.Properties == nil || workspace.Properties.ManagedVirtualNetwork == nil {
		return nil
	}
	vnetName := *workspace.Properties.ManagedVirtualNetwork
	if vnetName == "" {
		return nil
	}

	_, _, credential, clientOptions := az.getClientArgs()
	ctx := context.Background()

	armEndpoint := "https://management.azure.com"
	armAudience := armEndpoint
	if ep, ok := clientOptions.Cloud.Services[cloud.ResourceManager]; ok {
		armEndpoint = ep.Endpoint
		if ep.Audience != "" {
			armAudience = ep.Audience
		}
	}

	token, err := credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{armAudience + "/.default"},
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s%s/managedVirtualNetworks/%s/managedPrivateEndpoints?api-version=2021-06-01",
		armEndpoint, *workspace.ID, vnetName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("listing managed private endpoints: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Value []struct {
			ID   *string `json:"id"`
			Name *string `json:"name"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	for _, item := range result.Value {
		az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_synapse_managed_private_endpoint")
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
		err = az.appendManagedPrivateEndpoint(workspace)
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
