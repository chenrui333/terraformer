// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

func initDashboardResources(
	service *DatadogService,
	serviceName string,
	createResource func(string) terraformutils.Resource,
	createResources func([]datadogV1.DashboardSummaryDefinition) []terraformutils.Resource,
) error {
	datadogClient := service.Args["datadogClient"].(*datadog.APIClient)
	auth := service.Args["auth"].(context.Context)
	api := datadogV1.NewDashboardsApi(datadogClient)

	resources := []terraformutils.Resource{}
	for _, filter := range service.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}

		for _, value := range filter.AcceptableValues {
			dashboard, httpResp, err := api.GetDashboard(auth, value)
			closeDatadogResponseBody(httpResp)
			if err != nil {
				return err
			}

			resources = append(resources, createResource(dashboard.GetId()))
		}
	}

	if len(resources) > 0 {
		service.Resources = resources
		return nil
	}

	dashboards, err := listDatadogDashboards(auth, api)
	if err != nil {
		return err
	}
	service.Resources = createResources(dashboards)
	return nil
}

func listDatadogDashboards(auth context.Context, api *datadogV1.DashboardsApi) ([]datadogV1.DashboardSummaryDefinition, error) {
	pageSize := int64(100)
	items, cancel := api.ListDashboardsWithPagination(auth, *datadogV1.NewListDashboardsOptionalParameters().WithCount(pageSize))
	defer cancel()

	dashboards := []datadogV1.DashboardSummaryDefinition{}
	for item := range items {
		if item.Error != nil {
			return nil, item.Error
		}
		dashboards = append(dashboards, item.Item)
	}

	return dashboards, nil
}
