// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const datadogDatastoreServiceName = "datastore"

var (
	// DatastoreAllowEmptyValues ...
	DatastoreAllowEmptyValues = []string{"description"}
)

// DatastoreGenerator ...
type DatastoreGenerator struct {
	DatadogService
}

func (g *DatastoreGenerator) createResource(datastoreID string) (terraformutils.Resource, error) {
	return newDatadogIDResource(datadogDatastoreServiceName, datastoreID, DatastoreAllowEmptyValues)
}

func (g *DatastoreGenerator) createResources(datastoreIDs []string) ([]terraformutils.Resource, error) {
	return datadogIDResources(datadogDatastoreServiceName, datastoreIDs, DatastoreAllowEmptyValues)
}

// InitResources Generate TerraformResources from Datadog API,
// from each datastore create 1 TerraformResource.
func (g *DatastoreGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewActionsDatastoresApi(datadogClient)

	resources, hasIDFilter, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if hasIDFilter {
		g.Resources = resources
		return nil
	}

	datastoreIDs, err := g.listDatastoreIDs(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(datastoreIDs)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *DatastoreGenerator) filteredResources(auth context.Context, api *datadogV2.ActionsDatastoresApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || filter.ServiceName != datadogDatastoreServiceName {
			continue
		}
		hasIDFilter = true
		for _, value := range filter.AcceptableValues {
			datastoreID, err := g.getDatastoreID(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(datastoreID)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, hasIDFilter, nil
}

func (g *DatastoreGenerator) getDatastoreID(auth context.Context, api *datadogV2.ActionsDatastoresApi, datastoreID string) (string, error) {
	resp, httpResp, err := api.GetDatastore(auth, datastoreID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return "", err
	}
	data := resp.GetData()
	responseID := data.GetId()
	if responseID == "" {
		return datastoreID, nil
	}
	return responseID, nil
}

func (g *DatastoreGenerator) listDatastoreIDs(auth context.Context, api *datadogV2.ActionsDatastoresApi) ([]string, error) {
	resp, httpResp, err := api.ListDatastores(auth)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, datastore := range resp.GetData() {
		datastoreID := datastore.GetId()
		if datastoreID == "" {
			continue
		}
		ids = append(ids, datastoreID)
	}
	return ids, nil
}
