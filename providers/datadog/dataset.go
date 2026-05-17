// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	datadogDatasetServiceName = "dataset"

	datasetProductFiltersKey = "product_filters"
)

var (
	// DatasetAllowEmptyValues ...
	DatasetAllowEmptyValues = []string{datasetProductFiltersKey + "."}
)

// DatasetGenerator ...
type DatasetGenerator struct {
	DatadogService
}

func (g *DatasetGenerator) createResource(datasetID string) (terraformutils.Resource, error) {
	return newDatadogIDResource(datadogDatasetServiceName, datasetID, DatasetAllowEmptyValues)
}

func (g *DatasetGenerator) createResources(datasetIDs []string) ([]terraformutils.Resource, error) {
	return datadogIDResources(datadogDatasetServiceName, datasetIDs, DatasetAllowEmptyValues)
}

func (g *DatasetGenerator) PostConvertHook() error {
	for i := range g.Resources {
		if err := preserveDatasetEmptyProductFilters(&g.Resources[i]); err != nil {
			return err
		}
	}
	return nil
}

func preserveDatasetEmptyProductFilters(resource *terraformutils.Resource) error {
	hasEmptyProductFilters, err := datasetStateHasEmptyProductFilters(resource)
	if err != nil {
		return err
	}
	if !hasEmptyProductFilters {
		return nil
	}
	if resource.Item == nil {
		resource.Item = map[string]interface{}{}
	}
	if value, ok := resource.Item[datasetProductFiltersKey]; !ok || !datasetValueHasValue(value) {
		resource.Item[datasetProductFiltersKey] = []interface{}{}
	}
	return preserveDatasetEmptyProductFiltersState(resource)
}

func datasetStateHasEmptyProductFilters(resource *terraformutils.Resource) (bool, error) {
	if resource == nil || resource.InstanceState == nil {
		return false, nil
	}
	if resource.InstanceState.Attributes != nil {
		if count, ok := resource.InstanceState.Attributes[datasetProductFiltersKey+".#"]; ok && count == "0" {
			return true, nil
		}
	}
	if len(resource.InstanceState.TypedAttributes) == 0 {
		return false, nil
	}
	typedAttributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(resource.InstanceState.TypedAttributes, &typedAttributes); err != nil {
		return false, err
	}
	rawValue, ok := typedAttributes[datasetProductFiltersKey]
	if !ok {
		return false, nil
	}
	return datasetRawMessageIsEmptyList(rawValue)
}

func preserveDatasetEmptyProductFiltersState(resource *terraformutils.Resource) error {
	if resource == nil || resource.InstanceState == nil {
		return nil
	}
	if resource.InstanceState.Attributes == nil {
		resource.InstanceState.Attributes = map[string]string{}
	}
	resource.InstanceState.Attributes[datasetProductFiltersKey+".#"] = "0"
	if len(resource.InstanceState.TypedAttributes) == 0 {
		return nil
	}

	typedAttributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(resource.InstanceState.TypedAttributes, &typedAttributes); err != nil {
		return err
	}
	rawValue, ok := typedAttributes[datasetProductFiltersKey]
	if ok {
		emptyList, err := datasetRawMessageIsEmptyList(rawValue)
		if err != nil {
			return err
		}
		if !emptyList {
			return nil
		}
	}
	typedAttributes[datasetProductFiltersKey] = json.RawMessage("[]")
	rawAttributes, err := json.Marshal(typedAttributes)
	if err != nil {
		return err
	}
	resource.InstanceState.SetTypedAttributes(rawAttributes)
	return nil
}

func datasetRawMessageIsEmptyList(rawValue json.RawMessage) (bool, error) {
	if len(bytes.TrimSpace(rawValue)) == 0 {
		return false, nil
	}
	var value []interface{}
	if err := json.Unmarshal(rawValue, &value); err != nil {
		return false, err
	}
	return len(value) == 0, nil
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

	datasetIDs, err := g.listDatasetIDs(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(datasetIDs)
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
			datasetID, err := g.getDatasetID(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(datasetID)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, hasIDFilter, nil
}

func (g *DatasetGenerator) getDatasetID(auth context.Context, api *datadogV2.DatasetsApi, datasetID string) (string, error) {
	resp, httpResp, err := api.GetDataset(auth, datasetID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return "", err
	}
	data := resp.GetData()
	responseID := data.GetId()
	if responseID == "" {
		return datasetID, nil
	}
	return responseID, nil
}

func (g *DatasetGenerator) listDatasetIDs(auth context.Context, api *datadogV2.DatasetsApi) ([]string, error) {
	resp, httpResp, err := api.GetAllDatasets(auth)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, dataset := range resp.GetData() {
		datasetID := dataset.GetId()
		if datasetID == "" {
			continue
		}
		ids = append(ids, datasetID)
	}
	return ids, nil
}
