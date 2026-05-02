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
	// SpansMetricAllowEmptyValues ...
	SpansMetricAllowEmptyValues = []string{}
)

// SpansMetricGenerator ...
type SpansMetricGenerator struct {
	DatadogService
}

func (g *SpansMetricGenerator) createResources(spansMetrics []datadogV2.SpansMetricResponseData) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, spansMetric := range spansMetrics {
		resourceName := spansMetric.GetId()
		resources = append(resources, g.createResource(resourceName))
	}

	return resources
}

func (g *SpansMetricGenerator) createResource(spansMetricName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		spansMetricName,
		fmt.Sprintf("spans_metric_%s", spansMetricName),
		"datadog_spans_metric",
		"datadog",
		SpansMetricAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each span-based metric create 1 TerraformResource.
// Need SpansMetric Name as ID for terraform resource.
func (g *SpansMetricGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSpansMetricsApi(datadogClient)

	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("spans_metric") {
			hasIDFilter = true
			for _, value := range filter.AcceptableValues {
				spansMetric, httpResp, err := api.GetSpansMetric(auth, value)
				if httpResp != nil && httpResp.Body != nil {
					_ = httpResp.Body.Close()
				}
				if err != nil {
					return err
				}

				resources = append(resources, g.createResource(spansMetric.Data.GetId()))
			}
		}
	}

	if hasIDFilter || len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	spansMetrics, httpResp, err := api.ListSpansMetrics(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}
	g.Resources = g.createResources(spansMetrics.GetData())
	return nil
}
