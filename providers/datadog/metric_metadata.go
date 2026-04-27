// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"fmt"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// MetricMetadataAllowEmptyValues ...
	MetricMetadataAllowEmptyValues = []string{}
)

// MetricMetadataGenerator ...
type MetricMetadataGenerator struct {
	DatadogService
}

func (g *MetricMetadataGenerator) createResource(metricName string) terraformutils.Resource {
	return terraformutils.NewResource(
		metricName,
		fmt.Sprintf("metric_metadata_%s", metricName),
		"datadog_metric_metadata",
		"datadog",
		map[string]string{
			"metric": metricName,
		},
		MetricMetadataAllowEmptyValues,
		map[string]interface{}{},
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each metric create 1 TerraformResource.
// Need Metric Name as ID for terraform resource
func (g *MetricMetadataGenerator) InitResources() error {
	resources := []terraformutils.Resource{}
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("metric_metadata") {
			for _, value := range filter.AcceptableValues {
				resources = append(resources, g.createResource(value))
			}
		}
	}

	// Collecting all metrics_metadata can be an expensive task.
	// Hence, only allow collections of metrics passed via filter
	if len(resources) == 0 {
		log.Print("Filter(metric names as IDs) is required for importing datadog_metric_metadata resource")
		return nil
	}
	g.Resources = resources
	return nil
}
