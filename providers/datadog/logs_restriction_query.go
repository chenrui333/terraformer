// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// LogsRestrictionQueryAllowEmptyValues ...
	LogsRestrictionQueryAllowEmptyValues = []string{"restriction_query"}
)

// LogsRestrictionQueryGenerator ...
type LogsRestrictionQueryGenerator struct {
	DatadogService
}

func (g *LogsRestrictionQueryGenerator) createResource(queryID string) (terraformutils.Resource, error) {
	if queryID == "" {
		return terraformutils.Resource{}, fmt.Errorf("logs restriction query missing id")
	}

	return terraformutils.NewSimpleResource(
		queryID,
		fmt.Sprintf("logs_restriction_query_%s", queryID),
		"datadog_logs_restriction_query",
		"datadog",
		LogsRestrictionQueryAllowEmptyValues,
	), nil
}

func (g *LogsRestrictionQueryGenerator) createResources(queries []datadogV2.RestrictionQueryWithoutRelationships) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, query := range queries {
		resource, err := g.createResource(query.GetId())
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each logs_restriction_query create 1 TerraformResource.
func (g *LogsRestrictionQueryGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)

	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.ListRestrictionQueries", true)
	api := datadogV2.NewLogsRestrictionQueriesApi(datadogClient)

	// List all restriction queries with pagination
	var allItems []datadogV2.RestrictionQueryWithoutRelationships
	pageSize := int64(100)
	pageNumber := int64(0)
	remaining := int64(1)

	for remaining > int64(0) {
		optionalParams := datadogV2.NewListRestrictionQueriesOptionalParameters().
			WithPageSize(pageSize).
			WithPageNumber(pageNumber)

		resp, httpResp, err := api.ListRestrictionQueries(auth, *optionalParams)
		if httpResp != nil && httpResp.Body != nil {
			httpResp.Body.Close()
		}
		if err != nil {
			return err
		}
		items := resp.GetData()
		allItems = append(allItems, items...)

		if len(items) < int(pageSize) {
			break
		}
		pageNumber++
		remaining = int64(len(items))
	}

	resources, err := g.createResources(allItems)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
