// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// DashboardV2AllowEmptyValues ...
	DashboardV2AllowEmptyValues = []string{"tags.", "manage_status_definition.*.query"}
)

// DashboardV2Generator ...
type DashboardV2Generator struct {
	DatadogService
}

func (g *DashboardV2Generator) createResources(dashboards []datadogV1.DashboardSummaryDefinition) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, dashboard := range dashboards {
		resourceName := dashboard.GetId()
		resources = append(resources, g.createResource(resourceName))
	}

	return resources
}

func (g *DashboardV2Generator) createResource(dashboardID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		dashboardID,
		fmt.Sprintf("dashboard_v2_%s", dashboardID),
		"datadog_dashboard_v2",
		"datadog",
		DashboardV2AllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each dashboard_v2 create 1 TerraformResource.
// Need Dashboard ID as ID for terraform resource.
func (g *DashboardV2Generator) InitResources() error {
	return initDashboardResources(&g.DatadogService, "dashboard_v2", g.createResource, g.createResources)
}
