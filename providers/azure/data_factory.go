// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datafactory/armdatafactory/v9"
	"github.com/chenrui333/terraformer/terraformutils"
)

type DataFactoryGenerator struct {
	AzureService
}

// Maps item.Properties.Type -> terraform.ResourceType
// Information extracted from
//
//	SupportedResources are in:
//	@ github.com/azure/azure-sdk-for-go@v42.3.0+incompatible/services/datafactory/mgmt/2018-06-01/datafactory/models.go
//	PossibleTypeBasicDatasetValues, PossibleTypeBasicIntegrationRuntimeValues, PossibleTypeBasicLinkedServiceValues, PossibleTypeBasicTriggerValues
//	TypeBasicDataset,TypeBasicIntegrationRuntime, TypeBasicLinkedService, TypeBasicTrigger, TypeBasicDataFlow
var (
	SupportedResources = map[string]string{
		"AzureBlob":                "azurerm_data_factory_dataset_azure_blob",
		"Binary":                   "azurerm_data_factory_dataset_binary",
		"CosmosDbSqlApiCollection": "azurerm_data_factory_dataset_cosmosdb_sqlapi",
		"CustomDataset":            "azurerm_data_factory_custom_dataset",
		"DelimitedText":            "azurerm_data_factory_dataset_delimited_text",
		"HttpFile":                 "azurerm_data_factory_dataset_http",
		"Json":                     "azurerm_data_factory_dataset_json",
		"MySqlTable":               "azurerm_data_factory_dataset_mysql",
		"Parquet":                  "azurerm_data_factory_dataset_parquet",
		"PostgreSqlTable":          "azurerm_data_factory_dataset_postgresql",
		"SnowflakeTable":           "azurerm_data_factory_dataset_snowflake",
		"SqlServerTable":           "azurerm_data_factory_dataset_sql_server_table",
		"IntegrationRuntime":       "azurerm_data_factory_integration_runtime_azure",
		"Managed":                  "azurerm_data_factory_integration_runtime_azure_ssis",
		"SelfHosted":               "azurerm_data_factory_integration_runtime_self_hosted",
		"AzureBlobStorage":         "azurerm_data_factory_linked_service_azure_blob_storage",
		"AzureDatabricks":          "azurerm_data_factory_linked_service_azure_databricks",
		"AzureFileStorage":         "azurerm_data_factory_linked_service_azure_file_storage",
		"AzureFunction":            "azurerm_data_factory_linked_service_azure_function",
		"AzureSearch":              "azurerm_data_factory_linked_service_azure_search",
		"AzureSqlDatabase":         "azurerm_data_factory_linked_service_azure_sql_database",
		"AzureTableStorage":        "azurerm_data_factory_linked_service_azure_table_storage",
		"CosmosDb":                 "azurerm_data_factory_linked_service_cosmosdb",
		"CustomDataSource":         "azurerm_data_factory_linked_custom_service",
		"AzureBlobFS":              "azurerm_data_factory_linked_service_data_lake_storage_gen2",
		"AzureKeyVault":            "azurerm_data_factory_linked_service_key_vault",
		"AzureDataExplore":         "azurerm_data_factory_linked_service_kusto",
		"MySql":                    "azurerm_data_factory_linked_service_mysql",
		"OData":                    "azurerm_data_factory_linked_service_odata",
		"PostgreSql":               "azurerm_data_factory_linked_service_postgresql",
		"Sftp":                     "azurerm_data_factory_linked_service_sftp",
		"Snowflake":                "azurerm_data_factory_linked_service_snowflake",
		"SqlServer":                "azurerm_data_factory_linked_service_sql_server",
		"AzureSqlDW":               "azurerm_data_factory_linked_service_synapse",
		"Web":                      "azurerm_data_factory_linked_service_web",
		"BlobEventsTrigger":        "azurerm_data_factory_trigger_blob_event",
		"ScheduleTrigger":          "azurerm_data_factory_trigger_schedule",
		"TumblingWindowTrigger":    "azurerm_data_factory_trigger_tumbling_window",
	}
)

func getResourceTypeFrom(azureResourceName string) string {
	return SupportedResources[azureResourceName]
}

func (az *AzureService) appendResourceAs(resources []terraformutils.Resource, itemID string, itemName string, resourceType string, abbreviation string) []terraformutils.Resource {
	prefix := strings.ReplaceAll(resourceType, resourceType, abbreviation)
	suffix := strings.ReplaceAll(itemName, "-", "_")
	resourceName := prefix + "_" + suffix
	res := terraformutils.NewSimpleResource(itemID, resourceName, resourceType, az.ProviderName, []string{})
	resources = append(resources, res)
	return resources
}

func (az *DataFactoryGenerator) appendResourceFrom(resources []terraformutils.Resource, id string, name string, typeValue string) []terraformutils.Resource {
	if typeValue != "" {
		resourceType := getResourceTypeFrom(typeValue)
		if resourceType == "" {
			msg := fmt.Sprintf(`azurerm_data_factory: resource "%s" id: %s type: %s not handled yet by terraform or terraformer`, name, id, typeValue)
			log.Println(msg)
		} else {
			resources = az.appendResourceAs(resources, id, name, resourceType, "adf")
		}
	}
	return resources
}

func (az *DataFactoryGenerator) listFactories() ([]*armdatafactory.Factory, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armdatafactory.NewFactoriesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armdatafactory.Factory
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

func (az *DataFactoryGenerator) createDataFactoryResources(dataFactories []*armdatafactory.Factory) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for _, item := range dataFactories {
		resources = az.appendResourceAs(resources, *item.ID, *item.Name, "azurerm_data_factory", "adf")
	}
	return resources, nil
}

func (az *DataFactoryGenerator) createIntegrationRuntimesResources(dataFactories []*armdatafactory.Factory) ([]terraformutils.Resource, error) {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armdatafactory.NewIntegrationRuntimesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []terraformutils.Resource
	for _, factory := range dataFactories {
		id, err := ParseAzureResourceID(*factory.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByFactoryPager(id.ResourceGroup, *factory.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, item := range page.Value {
				resourceType := getIntegrationRuntimeType(item)
				resources = az.appendResourceAs(resources, *item.ID, *item.Name, resourceType, "adfr")
			}
		}
	}
	return resources, nil
}

func getIntegrationRuntimeType(item *armdatafactory.IntegrationRuntimeResource) string {
	if item.Properties == nil {
		return "azurerm_data_factory_integration_runtime_azure"
	}
	switch props := item.Properties.(type) {
	case *armdatafactory.SelfHostedIntegrationRuntime:
		_ = props
		return "azurerm_data_factory_integration_runtime_self_hosted"
	case *armdatafactory.ManagedIntegrationRuntime:
		if props.TypeProperties != nil && props.TypeProperties.SsisProperties != nil {
			return "azurerm_data_factory_integration_runtime_azure_ssis"
		}
		return "azurerm_data_factory_integration_runtime_azure"
	default:
		return "azurerm_data_factory_integration_runtime_azure"
	}
}

func (az *DataFactoryGenerator) createLinkedServiceResources(dataFactories []*armdatafactory.Factory) ([]terraformutils.Resource, error) {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armdatafactory.NewLinkedServicesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []terraformutils.Resource
	for _, factory := range dataFactories {
		id, err := ParseAzureResourceID(*factory.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByFactoryPager(id.ResourceGroup, *factory.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, item := range page.Value {
				typeValue := getLinkedServiceType(item)
				resources = az.appendResourceFrom(resources, *item.ID, *item.Name, typeValue)
			}
		}
	}
	return resources, nil
}

func getLinkedServiceType(item *armdatafactory.LinkedServiceResource) string {
	if item.Properties == nil {
		return ""
	}
	ls := item.Properties.GetLinkedService()
	if ls.Type == nil {
		return ""
	}
	return *ls.Type
}

func (az *DataFactoryGenerator) createPipelineResources(dataFactories []*armdatafactory.Factory) ([]terraformutils.Resource, error) {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armdatafactory.NewPipelinesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []terraformutils.Resource
	for _, factory := range dataFactories {
		id, err := ParseAzureResourceID(*factory.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByFactoryPager(id.ResourceGroup, *factory.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, item := range page.Value {
				resources = az.appendResourceAs(resources, *item.ID, *item.Name, "azurerm_data_factory_pipeline", "adfp")
			}
		}
	}
	return resources, nil
}

func (az *DataFactoryGenerator) createPipelineTriggerScheduleResources(dataFactories []*armdatafactory.Factory) ([]terraformutils.Resource, error) {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armdatafactory.NewTriggersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []terraformutils.Resource
	for _, factory := range dataFactories {
		id, err := ParseAzureResourceID(*factory.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByFactoryPager(id.ResourceGroup, *factory.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, item := range page.Value {
				typeValue := getTriggerType(item)
				resources = az.appendResourceFrom(resources, *item.ID, *item.Name, typeValue)
			}
		}
	}
	return resources, nil
}

func getTriggerType(item *armdatafactory.TriggerResource) string {
	if item.Properties == nil {
		return ""
	}
	t := item.Properties.GetTrigger()
	if t.Type == nil {
		return ""
	}
	return *t.Type
}

func (az *DataFactoryGenerator) createDataFlowResources(dataFactories []*armdatafactory.Factory) ([]terraformutils.Resource, error) {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armdatafactory.NewDataFlowsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []terraformutils.Resource
	for _, factory := range dataFactories {
		id, err := ParseAzureResourceID(*factory.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByFactoryPager(id.ResourceGroup, *factory.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, item := range page.Value {
				resources = az.appendResourceAs(resources, *item.ID, *item.Name, "azurerm_data_factory_data_flow", "adfl")
			}
		}
	}
	return resources, nil
}

func (az *DataFactoryGenerator) createPipelineDatasetResources(dataFactories []*armdatafactory.Factory) ([]terraformutils.Resource, error) {
	subscriptionID, _, credential, clientOptions := az.getClientArgs()
	client, err := armdatafactory.NewDatasetsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []terraformutils.Resource
	for _, factory := range dataFactories {
		id, err := ParseAzureResourceID(*factory.ID)
		if err != nil {
			return nil, err
		}
		pager := client.NewListByFactoryPager(id.ResourceGroup, *factory.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, item := range page.Value {
				typeValue := getDatasetType(item)
				resources = az.appendResourceFrom(resources, *item.ID, *item.Name, typeValue)
			}
		}
	}
	return resources, nil
}

func getDatasetType(item *armdatafactory.DatasetResource) string {
	if item.Properties == nil {
		return ""
	}
	ds := item.Properties.GetDataset()
	if ds.Type == nil {
		return ""
	}
	return *ds.Type
}

func (az *DataFactoryGenerator) InitResources() error {
	dataFactories, err := az.listFactories()
	if err != nil {
		return err
	}

	factoriesFunctions := []func([]*armdatafactory.Factory) ([]terraformutils.Resource, error){
		az.createDataFactoryResources,
		az.createIntegrationRuntimesResources,
		az.createLinkedServiceResources,
		az.createPipelineResources,
		az.createPipelineTriggerScheduleResources,
		az.createPipelineDatasetResources,
		az.createDataFlowResources,
	}

	for _, f := range factoriesFunctions {
		resources, ero := f(dataFactories)
		if ero != nil {
			return ero
		}
		az.Resources = append(az.Resources, resources...)
	}
	return nil
}

// PostGenerateHook for formatting json properties as heredoc
// - azurerm_data_factory_pipeline property activities_json
func (az *DataFactoryGenerator) PostConvertHook() error {
	for i, resource := range az.Resources {
		if resource.InstanceInfo.Type == "azurerm_data_factory_pipeline" {
			if val, ok := az.Resources[i].Item["activities_json"]; ok {
				if val != nil {
					json := val.(string)
					hereDoc := asHereDoc(json)
					az.Resources[i].Item["activities_json"] = hereDoc
				}
			}
		}
	}
	return nil
}
