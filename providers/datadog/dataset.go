// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const datadogDatasetServiceName = "dataset"

var (
	// DatasetAllowEmptyValues ...
	DatasetAllowEmptyValues = []string{}
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
