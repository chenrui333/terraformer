// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mariadb/armmariadb"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/chenrui333/terraformer/terraformutils"
)

type DatabasesGenerator struct {
	AzureService
}

// --- MariaDB ---

func (g *DatabasesGenerator) getMariaDBServers() ([]*armmariadb.Server, error) {
	ctx := context.Background()
	subscriptionID, resourceGroup, credential, clientOptions := g.getClientArgs()

	client, err := armmariadb.NewServersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var servers []*armmariadb.Server
	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			servers = append(servers, page.Value...)
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			servers = append(servers, page.Value...)
		}
	}

	return servers, nil
}

func (g *DatabasesGenerator) createMariaDBServerResources(servers []*armmariadb.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	for _, server := range servers {
		resources = append(resources, terraformutils.NewResource(
			*server.ID,
			*server.Name,
			"azurerm_mariadb_server",
			g.ProviderName,
			map[string]string{},
			[]string{},
			map[string]interface{}{
				"administrator_login_password": "",
			}))
	}

	return resources, nil
}

func (g *DatabasesGenerator) createMariaDBConfigurationResources(servers []*armmariadb.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armmariadb.NewConfigurationsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, config := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*config.ID,
					*config.Name+"-"+*server.Name,
					"azurerm_mariadb_configuration",
					g.ProviderName,
					[]string{"value"}))
			}
		}
	}

	return resources, nil
}

func (g *DatabasesGenerator) createMariaDBDatabaseResources(servers []*armmariadb.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armmariadb.NewDatabasesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, database := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*database.ID,
					*database.Name+"-"+*server.Name,
					"azurerm_mariadb_database",
					g.ProviderName,
					[]string{}))
			}
		}
	}

	return resources, nil
}

func (g *DatabasesGenerator) createMariaDBFirewallRuleResources(servers []*armmariadb.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armmariadb.NewFirewallRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, rule := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*rule.ID,
					*rule.Name,
					"azurerm_mariadb_firewall_rule",
					g.ProviderName,
					[]string{}))
			}
		}
	}

	return resources, nil
}

func (g *DatabasesGenerator) createMariaDBVirtualNetworkRuleResources(servers []*armmariadb.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armmariadb.NewVirtualNetworkRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, rule := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*rule.ID,
					*rule.Name,
					"azurerm_mariadb_virtual_network_rule",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

// --- MySQL ---

func (g *DatabasesGenerator) getMySQLServers() ([]*armmysql.Server, error) {
	ctx := context.Background()
	subscriptionID, resourceGroup, credential, clientOptions := g.getClientArgs()

	client, err := armmysql.NewServersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var servers []*armmysql.Server
	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			servers = append(servers, page.Value...)
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			servers = append(servers, page.Value...)
		}
	}

	return servers, nil
}

func (g *DatabasesGenerator) createMySQLServerResources(servers []*armmysql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	for _, server := range servers {
		resources = append(resources, terraformutils.NewResource(
			*server.ID,
			*server.Name,
			"azurerm_mysql_server",
			g.ProviderName,
			map[string]string{},
			[]string{},
			map[string]interface{}{
				"administrator_login_password": "",
			}))
	}

	return resources, nil
}

func (g *DatabasesGenerator) createMySQLConfigurationResources(servers []*armmysql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armmysql.NewConfigurationsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, config := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*config.ID,
					*config.Name+"-"+*server.Name,
					"azurerm_mysql_configuration",
					g.ProviderName,
					[]string{"value"}))
			}
		}
	}

	return resources, nil
}

func (g *DatabasesGenerator) createMySQLDatabaseResources(servers []*armmysql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armmysql.NewDatabasesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, database := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*database.ID,
					*database.Name+"-"+*server.Name,
					"azurerm_mysql_database",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) createMySQLFirewallRuleResources(servers []*armmysql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armmysql.NewFirewallRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, rule := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*rule.ID,
					*rule.Name,
					"azurerm_mysql_firewall_rule",
					g.ProviderName,
					[]string{}))
			}
		}
	}

	return resources, nil
}

func (g *DatabasesGenerator) createMySQLVirtualNetworkRuleResources(servers []*armmysql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armmysql.NewVirtualNetworkRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, rule := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*rule.ID,
					*rule.Name,
					"azurerm_mysql_virtual_network_rule",
					g.ProviderName,
					[]string{}))
			}
		}
	}

	return resources, nil
}

// --- PostgreSQL ---

func (g *DatabasesGenerator) getPostgreSQLServers() ([]*armpostgresql.Server, error) {
	ctx := context.Background()
	subscriptionID, resourceGroup, credential, clientOptions := g.getClientArgs()

	client, err := armpostgresql.NewServersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var servers []*armpostgresql.Server
	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			servers = append(servers, page.Value...)
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			servers = append(servers, page.Value...)
		}
	}

	return servers, nil
}

func (g *DatabasesGenerator) createPostgreSQLServerResources(servers []*armpostgresql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	for _, server := range servers {
		resources = append(resources, terraformutils.NewResource(
			*server.ID,
			*server.Name,
			"azurerm_postgresql_server",
			g.ProviderName,
			map[string]string{},
			[]string{},
			map[string]interface{}{
				"administrator_login_password": "",
			}))
	}

	return resources, nil
}

func (g *DatabasesGenerator) createPostgreSQLDatabaseResources(servers []*armpostgresql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armpostgresql.NewDatabasesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, database := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*database.ID,
					*database.Name+"-"+*server.Name,
					"azurerm_postgresql_database",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) createPostgreSQLConfigurationResources(servers []*armpostgresql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armpostgresql.NewConfigurationsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, config := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*config.ID,
					*config.Name+"-"+*server.Name,
					"azurerm_postgresql_configuration",
					g.ProviderName,
					[]string{"value"}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) createPostgreSQLFirewallRuleResources(servers []*armpostgresql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armpostgresql.NewFirewallRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, rule := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*rule.ID,
					*rule.Name,
					"azurerm_postgresql_firewall_rule",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) createPostgreSQLVirtualNetworkRuleResources(servers []*armpostgresql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armpostgresql.NewVirtualNetworkRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, rule := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*rule.ID,
					*rule.Name,
					"azurerm_postgresql_virtual_network_rule",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

// --- SQL Server (MSSQL) ---

func (g *DatabasesGenerator) getSQLServers() ([]*armsql.Server, error) {
	ctx := context.Background()
	subscriptionID, resourceGroup, credential, clientOptions := g.getClientArgs()

	client, err := armsql.NewServersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var servers []*armsql.Server
	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			servers = append(servers, page.Value...)
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			servers = append(servers, page.Value...)
		}
	}

	return servers, nil
}

func (g *DatabasesGenerator) createSQLServerResources(servers []*armsql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	for _, server := range servers {
		resources = append(resources, terraformutils.NewResource(
			*server.ID,
			*server.Name,
			"azurerm_mssql_server",
			g.ProviderName,
			map[string]string{},
			[]string{},
			map[string]interface{}{
				"administrator_login_password": "",
			}))
	}

	return resources, nil
}

func (g *DatabasesGenerator) createSQLDatabaseResources(servers []*armsql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armsql.NewDatabasesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, database := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*database.ID,
					*database.Name+"-"+*server.Name,
					"azurerm_mssql_database",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) createSQLFirewallRuleResources(servers []*armsql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armsql.NewFirewallRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, rule := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*rule.ID,
					*rule.Name,
					"azurerm_mssql_firewall_rule",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) createSQLVirtualNetworkRuleResources(servers []*armsql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armsql.NewVirtualNetworkRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, rule := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*rule.ID,
					*rule.Name,
					"azurerm_sql_virtual_network_rule",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) createSQLElasticPoolResources(servers []*armsql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armsql.NewElasticPoolsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, pool := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*pool.ID,
					*pool.Name,
					"azurerm_sql_elasticpool",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) createSQLFailoverResources(servers []*armsql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armsql.NewFailoverGroupsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, failoverGroup := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*failoverGroup.ID,
					*failoverGroup.Name,
					"azurerm_sql_failover_group",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) createSQLADAdministratorResources(servers []*armsql.Server) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID, _, credential, clientOptions := g.getClientArgs()

	client, err := armsql.NewServerAzureADAdministratorsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		id, err := ParseAzureResourceID(*server.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByServerPager(id.ResourceGroup, *server.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, administrator := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*administrator.ID,
					*administrator.Name,
					"azurerm_sql_active_directory_administrator",
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *DatabasesGenerator) InitResources() error {
	mariadbServers, err := g.getMariaDBServers()
	if err != nil {
		return err
	}

	mysqlServers, err := g.getMySQLServers()
	if err != nil {
		return err
	}

	postgresqlServers, err := g.getPostgreSQLServers()
	if err != nil {
		return err
	}

	sqlServers, err := g.getSQLServers()
	if err != nil {
		return err
	}

	mariadbFunctions := []func([]*armmariadb.Server) ([]terraformutils.Resource, error){
		g.createMariaDBServerResources,
		g.createMariaDBDatabaseResources,
		g.createMariaDBConfigurationResources,
		g.createMariaDBFirewallRuleResources,
		g.createMariaDBVirtualNetworkRuleResources,
	}

	mysqlFunctions := []func([]*armmysql.Server) ([]terraformutils.Resource, error){
		g.createMySQLServerResources,
		g.createMySQLDatabaseResources,
		g.createMySQLConfigurationResources,
		g.createMySQLFirewallRuleResources,
		g.createMySQLVirtualNetworkRuleResources,
	}

	postgresqlFunctions := []func([]*armpostgresql.Server) ([]terraformutils.Resource, error){
		g.createPostgreSQLServerResources,
		g.createPostgreSQLDatabaseResources,
		g.createPostgreSQLConfigurationResources,
		g.createPostgreSQLFirewallRuleResources,
		g.createPostgreSQLVirtualNetworkRuleResources,
	}

	sqlFunctions := []func([]*armsql.Server) ([]terraformutils.Resource, error){
		g.createSQLServerResources,
		g.createSQLDatabaseResources,
		g.createSQLADAdministratorResources,
		g.createSQLElasticPoolResources,
		g.createSQLFailoverResources,
		g.createSQLFirewallRuleResources,
		g.createSQLVirtualNetworkRuleResources,
	}

	for _, f := range mariadbFunctions {
		resources, err := f(mariadbServers)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	for _, f := range mysqlFunctions {
		resources, err := f(mysqlServers)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	for _, f := range postgresqlFunctions {
		resources, err := f(postgresqlServers)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	for _, f := range sqlFunctions {
		resources, err := f(sqlServers)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	return nil
}

func (g *DatabasesGenerator) PostConvertHook() error {
	dbEngines := []string{
		"mariadb",
		"mysql",
		"postgresql",
		"sql",
	}

	for _, engineName := range dbEngines {
		for _, resource := range g.Resources {
			dbServerResourceType := fmt.Sprintf("azurerm_%s_server", engineName)
			if resource.InstanceInfo.Type == dbServerResourceType {
				dbName := resource.Item["name"]
				for rIdx, r := range g.Resources {
					if r.InstanceInfo.Type != dbServerResourceType &&
						strings.Contains(r.InstanceInfo.Type, engineName) &&
						r.Item["server_name"] == dbName {
						g.Resources[rIdx].Item["server_name"] = fmt.Sprintf("${%s.%s}", resource.InstanceInfo.Id, "name")
					}
				}
			}
		}
	}

	return nil
}
