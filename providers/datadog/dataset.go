// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	datadogDatasetServiceName = "dataset"

	datasetPrincipalsKey     = "principals"
	datasetProductFiltersKey = "product_filters"
	datasetFiltersKey        = "filters"
)

var (
	// DatasetAllowEmptyValues ...
	DatasetAllowEmptyValues = []string{
		datasetPrincipalsKey,
		"product_filters\\.[0-9]+\\.filters",
	}
)

// DatasetGenerator ...
type DatasetGenerator struct {
	DatadogService
}

func (g *DatasetGenerator) createResource(dataset datadogV2.DatasetResponse) (terraformutils.Resource, error) {
	datasetID := dataset.GetId()
	if datasetID == "" {
		return terraformutils.Resource{}, fmt.Errorf("dataset missing id")
	}

	attributes := dataset.GetAttributes()
	name, ok := attributes.GetNameOk()
	if !ok || name == nil {
		return terraformutils.Resource{}, fmt.Errorf("dataset %s missing required name", datasetID)
	}
	principals, ok := attributes.GetPrincipalsOk()
	if !ok || principals == nil {
		return terraformutils.Resource{}, fmt.Errorf("dataset %s missing required principals", datasetID)
	}

	stateAttributes := datasetStateAttributes(datasetID, *name, *principals)
	if productFilters, ok := attributes.GetProductFiltersOk(); ok && productFilters != nil {
		addDatasetProductFilterAttributes(stateAttributes, *productFilters)
	}

	return terraformutils.NewResource(
		datasetID,
		fmt.Sprintf("dataset_%s", datasetID),
		"datadog_dataset",
		"datadog",
		stateAttributes,
		DatasetAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

func (g *DatasetGenerator) createResources(datasets []datadogV2.DatasetResponse) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, dataset := range datasets {
		resource, err := g.createResource(dataset)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func datasetStateAttributes(datasetID, name string, principals []string) map[string]string {
	attributes := map[string]string{
		"id":                        datasetID,
		"name":                      name,
		datasetPrincipalsKey + ".#": strconv.Itoa(len(principals)),
	}
	for index, principal := range principals {
		attributes[fmt.Sprintf("%s.%d", datasetPrincipalsKey, index)] = principal
	}
	return attributes
}

func addDatasetProductFilterAttributes(attributes map[string]string, productFilters []datadogV2.FiltersPerProduct) {
	if len(productFilters) == 0 {
		return
	}

	attributes[datasetProductFiltersKey+".#"] = strconv.Itoa(len(productFilters))
	for productFilterIndex, productFilter := range productFilters {
		prefix := fmt.Sprintf("%s.%d", datasetProductFiltersKey, productFilterIndex)
		attributes[prefix+".product"] = productFilter.GetProduct()
		filters := productFilter.GetFilters()
		attributes[prefix+"."+datasetFiltersKey+".#"] = strconv.Itoa(len(filters))
		for filterIndex, filter := range filters {
			attributes[fmt.Sprintf("%s.%s.%d", prefix, datasetFiltersKey, filterIndex)] = filter
		}
	}
}

func (g *DatasetGenerator) PostConvertHook() error {
	for i := range g.Resources {
		preserveDatasetEmptyPrincipals(&g.Resources[i])
		preserveDatasetEmptyProductFilterFilters(&g.Resources[i])
	}
	return nil
}

func preserveDatasetEmptyPrincipals(resource *terraformutils.Resource) {
	if resource == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return
	}
	if resource.InstanceState.Attributes[datasetPrincipalsKey+".#"] != "0" {
		return
	}
	if resource.Item == nil {
		resource.Item = map[string]interface{}{}
	}
	if value, ok := resource.Item[datasetPrincipalsKey]; ok && datasetValueHasValue(value) {
		return
	}
	resource.Item[datasetPrincipalsKey] = []interface{}{}
}

func preserveDatasetEmptyProductFilterFilters(resource *terraformutils.Resource) {
	if resource == nil || resource.Item == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return
	}
	productFilters, ok := resource.Item[datasetProductFiltersKey].([]interface{})
	if !ok {
		return
	}

	for index, productFilter := range productFilters {
		if resource.InstanceState.Attributes[fmt.Sprintf("%s.%d.%s.#", datasetProductFiltersKey, index, datasetFiltersKey)] != "0" {
			continue
		}
		filterBlock, ok := productFilter.(map[string]interface{})
		if !ok {
			continue
		}
		if value, ok := filterBlock[datasetFiltersKey]; ok && datasetValueHasValue(value) {
			continue
		}
		filterBlock[datasetFiltersKey] = []interface{}{}
	}
}

func datasetValueHasValue(value interface{}) bool {
	switch typedValue := value.(type) {
	case nil:
		return false
	case []interface{}:
		return len(typedValue) > 0
	default:
		return true
	}
}

// InitResources Generate TerraformResources from Datadog API,
// from each dataset create 1 TerraformResource.
func (g *DatasetGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.GetAllDatasets", true)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.GetDataset", true)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewDatasetsApi(datadogClient)

	resources, hasIDFilter, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if hasIDFilter {
		g.Resources = resources
		return nil
	}

	datasets, err := g.listDatasets(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(datasets)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *DatasetGenerator) filteredResources(auth context.Context, api *datadogV2.DatasetsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || filter.ServiceName != datadogDatasetServiceName {
			continue
		}
		hasIDFilter = true
		for _, value := range filter.AcceptableValues {
			dataset, err := g.getDataset(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(dataset)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, hasIDFilter, nil
}

func (g *DatasetGenerator) getDataset(auth context.Context, api *datadogV2.DatasetsApi, datasetID string) (datadogV2.DatasetResponse, error) {
	resp, httpResp, err := api.GetDataset(auth, datasetID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return datadogV2.DatasetResponse{}, err
	}
	return resp.GetData(), nil
}

func (g *DatasetGenerator) listDatasets(auth context.Context, api *datadogV2.DatasetsApi) ([]datadogV2.DatasetResponse, error) {
	resp, httpResp, err := api.GetAllDatasets(auth)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return nil, err
	}
	return resp.GetData(), nil
}
