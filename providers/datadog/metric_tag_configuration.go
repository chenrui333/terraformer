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
	// MetricTagConfigurationAllowEmptyValues ...
	MetricTagConfigurationAllowEmptyValues = []string{"tags."}
)

// Avoid Datadog's one-hour default window when importing all configured metric tags.
const metricTagConfigurationListWindowSeconds int64 = 60 * 60 * 24 * 30

// MetricTagConfigurationGenerator ...
type MetricTagConfigurationGenerator struct {
	DatadogService
}

func (g *MetricTagConfigurationGenerator) createResources(metricTagConfigurations []datadogV2.MetricTagConfiguration) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, metricTagConfiguration := range metricTagConfigurations {
		resource, err := g.createResource(metricTagConfiguration)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *MetricTagConfigurationGenerator) createResource(metricTagConfiguration datadogV2.MetricTagConfiguration) (terraformutils.Resource, error) {
	metricName := metricTagConfiguration.GetId()
	if metricName == "" {
		return terraformutils.Resource{}, fmt.Errorf("metric tag configuration missing metric name")
	}

	return terraformutils.NewSimpleResource(
		metricName,
		fmt.Sprintf("metric_tag_configuration_%s", metricName),
		"datadog_metric_tag_configuration",
		"datadog",
		MetricTagConfigurationAllowEmptyValues,
	), nil
}

func (g *MetricTagConfigurationGenerator) PostConvertHook() error {
	for i := range g.Resources {
		resource := &g.Resources[i]
		if resource.Item == nil {
			resource.Item = map[string]interface{}{}
		}
		if _, ok := resource.Item["tags"]; ok {
			continue
		}
		if !metricTagConfigurationStateHasEmptyTags(resource) {
			continue
		}
		resource.Item["tags"] = []interface{}{}
	}
	return nil
}

func metricTagConfigurationStateHasEmptyTags(resource *terraformutils.Resource) bool {
	if resource == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return false
	}
	return resource.InstanceState.Attributes["tags.#"] == "0"
}

// InitResources Generate TerraformResources from Datadog API,
// from each metric tag configuration create 1 TerraformResource.
func (g *MetricTagConfigurationGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewMetricsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	metricTagConfigurations, err := listMetricTagConfigurations(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(metricTagConfigurations)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *MetricTagConfigurationGenerator) filteredResources(auth context.Context, api *datadogV2.MetricsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("metric_tag_configuration") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			metricTagConfiguration, err := getMetricTagConfiguration(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(metricTagConfiguration)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getMetricTagConfiguration(auth context.Context, api *datadogV2.MetricsApi, metricName string) (datadogV2.MetricTagConfiguration, error) {
	response, httpResponse, err := api.ListTagConfigurationByName(auth, metricName)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.MetricTagConfiguration{}, err
	}

	if configuration, ok := response.GetDataOk(); ok {
		return *configuration, nil
	}
	if configuration, ok := metricTagConfigurationFromRawData(response.UnparsedObject["data"]); ok {
		return configuration, nil
	}

	return datadogV2.MetricTagConfiguration{}, fmt.Errorf("metric tag configuration %q not found", metricName)
}

func listMetricTagConfigurations(auth context.Context, api *datadogV2.MetricsApi) ([]datadogV2.MetricTagConfiguration, error) {
	metricTagConfigurations := []datadogV2.MetricTagConfiguration{}
	const pageSize int32 = 1000
	var cursor string

	for {
		optionalParams := datadogV2.NewListTagConfigurationsOptionalParameters().
			WithFilterConfigured(true).
			WithWindowSeconds(metricTagConfigurationListWindowSeconds).
			WithPageSize(pageSize)
		if cursor != "" {
			optionalParams.WithPageCursor(cursor)
		}

		response, httpResponse, err := api.ListTagConfigurations(auth, *optionalParams)
		closeDatadogResponseBody(httpResponse)
		if err != nil {
			return nil, err
		}

		metricTagConfigurations = append(metricTagConfigurations, metricTagConfigurationsFromItems(response.GetData())...)
		if len(response.GetData()) == 0 {
			metricTagConfigurations = append(metricTagConfigurations, metricTagConfigurationsFromRawData(response.UnparsedObject["data"])...)
		}

		nextCursor := nextMetricTagConfigurationCursor(response)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return metricTagConfigurations, nil
}

func metricTagConfigurationsFromItems(items []datadogV2.MetricsAndMetricTagConfigurations) []datadogV2.MetricTagConfiguration {
	metricTagConfigurations := []datadogV2.MetricTagConfiguration{}
	for _, item := range items {
		if item.MetricTagConfiguration != nil {
			metricTagConfigurations = append(metricTagConfigurations, *item.MetricTagConfiguration)
			continue
		}
		if item.Metric != nil {
			metricTagConfiguration, ok := metricTagConfigurationFromMetricName(item.Metric.GetId())
			if !ok {
				continue
			}
			metricTagConfigurations = append(metricTagConfigurations, metricTagConfiguration)
			continue
		}
		metricTagConfiguration, ok := metricTagConfigurationFromRawData(item.UnparsedObject)
		if !ok {
			continue
		}
		metricTagConfigurations = append(metricTagConfigurations, metricTagConfiguration)
	}
	return metricTagConfigurations
}

func metricTagConfigurationsFromRawData(rawData interface{}) []datadogV2.MetricTagConfiguration {
	rawConfigurations, ok := rawData.([]interface{})
	if !ok {
		return nil
	}

	metricTagConfigurations := []datadogV2.MetricTagConfiguration{}
	for _, rawConfiguration := range rawConfigurations {
		metricTagConfiguration, ok := metricTagConfigurationFromRawData(rawConfiguration)
		if !ok {
			continue
		}
		metricTagConfigurations = append(metricTagConfigurations, metricTagConfiguration)
	}
	return metricTagConfigurations
}

func metricTagConfigurationFromRawData(rawData interface{}) (datadogV2.MetricTagConfiguration, bool) {
	rawConfiguration, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV2.MetricTagConfiguration{}, false
	}
	if rawType, ok := rawConfiguration["type"].(string); ok {
		switch rawType {
		case string(datadogV2.METRICTAGCONFIGURATIONTYPE_MANAGE_TAGS), string(datadogV2.METRICTYPE_METRICS):
		default:
			return datadogV2.MetricTagConfiguration{}, false
		}
	}

	rawMetricName, ok := rawConfiguration["id"].(string)
	if !ok {
		return datadogV2.MetricTagConfiguration{}, false
	}
	return metricTagConfigurationFromMetricName(rawMetricName)
}

func metricTagConfigurationFromMetricName(metricName string) (datadogV2.MetricTagConfiguration, bool) {
	if metricName == "" {
		return datadogV2.MetricTagConfiguration{}, false
	}
	metricTagConfiguration := datadogV2.NewMetricTagConfigurationWithDefaults()
	metricTagConfiguration.SetId(metricName)
	return *metricTagConfiguration, true
}

func nextMetricTagConfigurationCursor(response datadogV2.MetricsAndMetricTagConfigurationsResponse) string {
	meta, ok := response.GetMetaOk()
	if !ok {
		return ""
	}
	pagination, ok := meta.GetPaginationOk()
	if !ok {
		return ""
	}
	nextCursor, ok := pagination.GetNextCursorOk()
	if !ok || nextCursor == nil {
		return ""
	}
	return *nextCursor
}
