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
	// RumMetricAllowEmptyValues ...
	RumMetricAllowEmptyValues = []string{}
)

// RumMetricGenerator ...
type RumMetricGenerator struct {
	DatadogService
}

func (g *RumMetricGenerator) createResources(rumMetrics []datadogV2.RumMetricResponseData) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, rumMetric := range rumMetrics {
		resourceName := rumMetric.GetId()
		resources = append(resources, g.createResource(resourceName))
	}

	return resources
}

func (g *RumMetricGenerator) createResource(rumMetricName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		rumMetricName,
		fmt.Sprintf("rum_metric_%s", rumMetricName),
		"datadog_rum_metric",
		"datadog",
		RumMetricAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each RUM-based metric create 1 TerraformResource.
// Need RumMetric Name as ID for terraform resource.
func (g *RumMetricGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewRumMetricsApi(datadogClient)

	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("rum_metric") {
			hasIDFilter = true
			for _, value := range filter.AcceptableValues {
				rumMetric, httpResp, err := api.GetRumMetric(auth, value)
				if httpResp != nil && httpResp.Body != nil {
					_ = httpResp.Body.Close()
				}
				if err != nil {
					return err
				}

				resources = append(resources, g.createResource(rumMetric.Data.GetId()))
			}
		}
	}

	if hasIDFilter || len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	rumMetrics, httpResp, err := api.ListRumMetrics(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}
	g.Resources = g.createResources(rumMetrics.GetData())
	return nil
}
