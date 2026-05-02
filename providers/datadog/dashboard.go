// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// DashboardAllowEmptyValues ...
	DashboardAllowEmptyValues = []string{"tags.", "manage_status_definition.*.query"}
)

// DashboardGenerator ...
type DashboardGenerator struct {
	DatadogService
}

func (g *DashboardGenerator) createResources(dashboards []datadogV1.DashboardSummaryDefinition) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, dashboard := range dashboards {
		resourceName := dashboard.GetId()
		resources = append(resources, g.createResource(resourceName))
	}

	return resources
}

func (g *DashboardGenerator) createResource(dashboardID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		dashboardID,
		fmt.Sprintf("dashboard_%s", dashboardID),
		"datadog_dashboard",
		"datadog",
		DashboardAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each dashboard create 1 TerraformResource.
// Need Dashboard ID as ID for terraform resource
func (g *DashboardGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV1.NewDashboardsApi(datadogClient)

	resources := []terraformutils.Resource{}
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("dashboard") {
			for _, value := range filter.AcceptableValues {
				dashboard, httpResp, err := api.GetDashboard(auth, value)
				if httpResp != nil && httpResp.Body != nil {
					_ = httpResp.Body.Close()
				}
				if err != nil {
					return err
				}

				resources = append(resources, g.createResource(dashboard.GetId()))
			}
		}
	}

	if len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	summary, httpResp, err := api.ListDashboards(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}
	g.Resources = g.createResources(summary.GetDashboards())
	return nil
}
