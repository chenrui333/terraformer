// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/databricks/armdatabricks"
)

type DatabricksGenerator struct {
	AzureService
}

func (az *DatabricksGenerator) listWorkspaces() ([]*armdatabricks.Workspace, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armdatabricks.NewWorkspacesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armdatabricks.Workspace
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

func (az *DatabricksGenerator) AppendWorkspace(workspace *armdatabricks.Workspace) {
	az.AppendSimpleResource(*workspace.ID, *workspace.Name, "azurerm_databricks_workspace")
}

func (az *DatabricksGenerator) InitResources() error {
	workspaces, err := az.listWorkspaces()
	if err != nil {
		return err
	}
	for _, workspace := range workspaces {
		az.AppendWorkspace(workspace)
	}
	return nil
}
