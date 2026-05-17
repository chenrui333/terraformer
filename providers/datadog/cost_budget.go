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
	CostBudgetAllowEmptyValues = []string{}
)

type CostBudgetGenerator struct {
	DatadogService
}

func (g *CostBudgetGenerator) createResource(budget datadogV2.Budget) terraformutils.Resource {
	id := budget.GetId()
	resourceName := fmt.Sprintf("cost_budget_%s", id)
	attrs := budget.GetAttributes()
	if name := attrs.GetName(); name != "" {
		resourceName = fmt.Sprintf("cost_budget_%s", name)
	}

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_cost_budget",
		"datadog",
		CostBudgetAllowEmptyValues,
	)
}

func (g *CostBudgetGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewCloudCostManagementApi(datadogClient)

	resp, httpResp, err := api.ListBudgets(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, budget := range resp.GetData() {
		resources = append(resources, g.createResource(budget))
	}
	g.Resources = resources
	return nil
}
