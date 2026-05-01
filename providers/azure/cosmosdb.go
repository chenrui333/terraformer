// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3"
	"github.com/chenrui333/terraformer/terraformutils"
)

type CosmosDBGenerator struct {
	AzureService
}

func (g *CosmosDBGenerator) listSQLDatabasesAndContainersBehind(resourceGroupName string, accountName string) ([]terraformutils.Resource, []terraformutils.Resource, error) {
	var resourcesDatabase []terraformutils.Resource
	var resourcesContainer []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armcosmos.NewSQLResourcesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, nil, err
	}

	pager := client.NewListSQLDatabasesPager(resourceGroupName, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, nil, err
		}
		for _, sqlDatabase := range page.Value {
			// NOTE:
			// For a similar reason as
			// https://github.com/terraform-providers/terraform-provider-azurerm/issues/7472#issuecomment-650684349
			// The cosmosdb resource format change is NOT yet addressed in terraform provider
			// This line is a workaround to convert to old format, and might be removed if they deprecate the old format
			sqlDatabaseIDInOldFormat := strings.Replace(*sqlDatabase.ID, "sqlDatabases", "databases", 1)
			resourcesDatabase = append(resourcesDatabase, terraformutils.NewSimpleResource(
				sqlDatabaseIDInOldFormat,
				*sqlDatabase.Name,
				"azurerm_cosmosdb_sql_database",
				g.ProviderName,
				[]string{}))

			containerPager := client.NewListSQLContainersPager(resourceGroupName, accountName, *sqlDatabase.Name, nil)
			for containerPager.More() {
				containerPage, err := containerPager.NextPage(ctx)
				if err != nil {
					return nil, nil, err
				}
				for _, sqlContainer := range containerPage.Value {
					// NOTE:
					// For a similar reason as
					// https://github.com/terraform-providers/terraform-provider-azurerm/issues/7472#issuecomment-650684349
					// The cosmosdb resource format change is NOT yet addressed in terraform provider
					// This line is a workaround to convert to old format, and might be removed if they deprecate the old format
					sqlContainerIDInOldFormat := strings.Replace(*sqlContainer.ID, "sqlDatabases", "databases", 1)
					resourcesContainer = append(resourcesContainer, terraformutils.NewSimpleResource(
						sqlContainerIDInOldFormat,
						*sqlContainer.Name,
						"azurerm_cosmosdb_sql_container",
						g.ProviderName,
						[]string{}))
				}
			}
		}
	}

	return resourcesDatabase, resourcesContainer, nil
}

func (g *CosmosDBGenerator) listTables(resourceGroupName string, accountName string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armcosmos.NewTableResourcesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	pager := client.NewListTablesPager(resourceGroupName, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, table := range page.Value {
			resources = append(resources, terraformutils.NewSimpleResource(
				*table.ID,
				*table.Name,
				"azurerm_cosmosdb_table",
				g.ProviderName,
				[]string{}))
		}
	}

	return resources, nil
}

func (g *CosmosDBGenerator) listAndAddForDatabaseAccounts() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, resourceGroup, credential, clientOptions := g.getClientArgs()

	client, err := armcosmos.NewDatabaseAccountsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var accounts []*armcosmos.DatabaseAccountGetResults
	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, page.Value...)
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, page.Value...)
		}
	}

	for _, account := range accounts {
		resources = append(resources, terraformutils.NewSimpleResource(
			*account.ID,
			*account.Name,
			"azurerm_cosmosdb_account",
			g.ProviderName,
			[]string{}))

		id, err := ParseAzureResourceID(*account.ID)
		if err != nil {
			return nil, err
		}

		tables, err := g.listTables(id.ResourceGroup, *account.Name)
		if err != nil {
			return nil, err
		}
		resources = append(resources, tables...)

		sqlDatabases, sqlContainers, err := g.listSQLDatabasesAndContainersBehind(id.ResourceGroup, *account.Name)
		if err != nil {
			return nil, err
		}
		resources = append(resources, sqlDatabases...)
		resources = append(resources, sqlContainers...)
	}

	return resources, nil
}

func (g *CosmosDBGenerator) InitResources() error {
	functions := []func() ([]terraformutils.Resource, error){
		g.listAndAddForDatabaseAccounts,
	}

	for _, f := range functions {
		resources, err := f()
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	return nil
}
