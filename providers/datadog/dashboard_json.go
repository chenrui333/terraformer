// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// DashboardJSONAllowEmptyValues ...
	DashboardJSONAllowEmptyValues = []string{"tags."}
)

// DashboardJSONGenerator ...
type DashboardJSONGenerator struct {
	DatadogService
}

func (g *DashboardJSONGenerator) createResources(dashboards []datadogV1.DashboardSummaryDefinition) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, dashboard := range dashboards {
		resourceName := dashboard.GetId()
		resources = append(resources, g.createResource(resourceName))
	}

	return resources
}

func (g *DashboardJSONGenerator) createResource(dashboardID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		dashboardID,
		fmt.Sprintf("dashboard_json_%s", dashboardID),
		"datadog_dashboard_json",
		"datadog",
		DashboardJSONAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each dashboard_json create 1 TerraformResource.
// Need Dashboard ID as ID for terraform resource
func (g *DashboardJSONGenerator) InitResources() error {
	return initDashboardResources(&g.DatadogService, "dashboard_json", g.createResource, g.createResources)
}
