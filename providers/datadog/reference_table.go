// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	datadogReferenceTableServiceName = "reference_table"
	datadogReferenceTablePageLimit   = int64(100)
)

var (
	// ReferenceTableAllowEmptyValues ...
	ReferenceTableAllowEmptyValues = []string{}
)

// ReferenceTableGenerator ...
type ReferenceTableGenerator struct {
	DatadogService
}

func (g *ReferenceTableGenerator) createResource(tableID string) (terraformutils.Resource, error) {
	return newDatadogIDResource(datadogReferenceTableServiceName, tableID, ReferenceTableAllowEmptyValues)
}

func (g *ReferenceTableGenerator) createResources(tableIDs []string) ([]terraformutils.Resource, error) {
	return datadogIDResources(datadogReferenceTableServiceName, tableIDs, ReferenceTableAllowEmptyValues)
}

// InitResources Generate TerraformResources from Datadog API,
// from each reference_table create 1 TerraformResource.
func (g *ReferenceTableGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewReferenceTablesApi(datadogClient)

	resources, hasIDFilter, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if hasIDFilter {
		g.Resources = resources
		return nil
	}

	tableIDs, err := g.listReferenceTableIDs(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(tableIDs)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *ReferenceTableGenerator) filteredResources(auth context.Context, api *datadogV2.ReferenceTablesApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || filter.ServiceName != datadogReferenceTableServiceName {
			continue
		}
		hasIDFilter = true
		for _, value := range filter.AcceptableValues {
			tableID, err := g.getReferenceTableID(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(tableID)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, hasIDFilter, nil
}

func (g *ReferenceTableGenerator) getReferenceTableID(auth context.Context, api *datadogV2.ReferenceTablesApi, tableID string) (string, error) {
	resp, httpResp, err := api.GetTable(auth, tableID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return "", err
	}
	data := resp.GetData()
	responseID := data.GetId()
	if responseID == "" {
		return tableID, nil
	}
	return responseID, nil
}

func (g *ReferenceTableGenerator) listReferenceTableIDs(auth context.Context, api *datadogV2.ReferenceTablesApi) ([]string, error) {
	ids := []string{}
	offset := int64(0)

	for {
		opts := datadogV2.NewListTablesOptionalParameters().
			WithPageLimit(datadogReferenceTablePageLimit).
			WithPageOffset(offset)

		resp, httpResp, err := api.ListTables(auth, *opts)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		tables := resp.GetData()
		for _, table := range tables {
			tableID := table.GetId()
			if tableID == "" {
				continue
			}
			ids = append(ids, tableID)
		}

		if int64(len(tables)) < datadogReferenceTablePageLimit {
			break
		}
		offset += datadogReferenceTablePageLimit
	}

	return ids, nil
}
