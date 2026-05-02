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
	// LogsMetricAllowEmptyValues ...
	LogsMetricAllowEmptyValues = []string{}
)

// LogsMetricGenerator ...
type LogsMetricGenerator struct {
	DatadogService
}

func (g *LogsMetricGenerator) createResources(logsMetrics []datadogV2.LogsMetricResponseData) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, logsMetric := range logsMetrics {
		resourceName := logsMetric.GetId()
		resources = append(resources, g.createResource(resourceName))
	}

	return resources
}

func (g *LogsMetricGenerator) createResource(logsMetricName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		logsMetricName,
		fmt.Sprintf("logs_metric_%s", logsMetricName),
		"datadog_logs_metric",
		"datadog",
		LogsMetricAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each log's metric create 1 TerraformResource.
// Need LogsMetric Name as ID for terraform resource
func (g *LogsMetricGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewLogsMetricsApi(datadogClient)

	resources := []terraformutils.Resource{}
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("logs_metric") {
			for _, value := range filter.AcceptableValues {
				logsMetric, httpResp, err := api.GetLogsMetric(auth, value)
				if httpResp != nil && httpResp.Body != nil {
					_ = httpResp.Body.Close()
				}
				if err != nil {
					return err
				}

				resources = append(resources, g.createResource(logsMetric.Data.GetId()))
			}
		}
	}

	if len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	logsMetrics, httpResp, err := api.ListLogsMetrics(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}
	g.Resources = g.createResources(logsMetrics.GetData())
	return nil
}
