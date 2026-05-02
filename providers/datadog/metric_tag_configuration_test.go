// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestMetricTagConfigurationCreateResource(t *testing.T) {
	metricTagConfiguration := datadogV2.NewMetricTagConfigurationWithDefaults()
	metricTagConfiguration.SetId("example.terraform.metric")

	generator := &MetricTagConfigurationGenerator{}
	resource, err := generator.createResource(*metricTagConfiguration)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "example.terraform.metric" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "example.terraform.metric")
	}
	if resource.ResourceName != "tfer--metric_tag_configuration_example-002E-terraform-002E-metric" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--metric_tag_configuration_example-002E-terraform-002E-metric")
	}
	if resource.InstanceInfo.Type != "datadog_metric_tag_configuration" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_metric_tag_configuration")
	}
}

func TestMetricTagConfigurationCreateResourceMissingMetricName(t *testing.T) {
	generator := &MetricTagConfigurationGenerator{}
	_, err := generator.createResource(datadogV2.MetricTagConfiguration{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing metric name error")
	}
}

func TestMetricTagConfigurationPostConvertHookPreservesEmptyTags(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"example.empty.tags.metric",
		"metric_tag_configuration_example.empty.tags.metric",
		"datadog_metric_tag_configuration",
		"datadog",
		MetricTagConfigurationAllowEmptyValues,
	)
	resource.InstanceState.Attributes = map[string]string{
		"metric_name": "example.empty.tags.metric",
		"tags.#":      "0",
	}
	resource.Item = map[string]interface{}{
		"metric_name": "example.empty.tags.metric",
	}

	generator := &MetricTagConfigurationGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	tags, ok := generator.Resources[0].Item["tags"].([]interface{})
	if !ok {
		t.Fatalf("tags = %T, want []interface{}", generator.Resources[0].Item["tags"])
	}
	if len(tags) != 0 {
		t.Fatalf("tags length = %d, want %d", len(tags), 0)
	}
}

func TestMetricTagConfigurationPostConvertHookDoesNotInventUnknownTags(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"example.unknown.tags.metric",
		"metric_tag_configuration_example.unknown.tags.metric",
		"datadog_metric_tag_configuration",
		"datadog",
		MetricTagConfigurationAllowEmptyValues,
	)
	resource.InstanceState.Attributes = map[string]string{
		"metric_name": "example.unknown.tags.metric",
	}
	resource.Item = map[string]interface{}{
		"metric_name": "example.unknown.tags.metric",
	}

	generator := &MetricTagConfigurationGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}
	if _, ok := generator.Resources[0].Item["tags"]; ok {
		t.Fatal("PostConvertHook added tags without empty tags state")
	}
}

func TestMetricTagConfigurationCreateResources(t *testing.T) {
	firstConfiguration := datadogV2.NewMetricTagConfigurationWithDefaults()
	firstConfiguration.SetId("example.first.metric")
	secondConfiguration := datadogV2.NewMetricTagConfigurationWithDefaults()
	secondConfiguration.SetId("example.second.metric")

	generator := &MetricTagConfigurationGenerator{}
	resources, err := generator.createResources([]datadogV2.MetricTagConfiguration{
		*firstConfiguration,
		*secondConfiguration,
	})
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}

	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(resources), 2)
	}
	if resources[0].ResourceName == resources[1].ResourceName {
		t.Fatalf("resource names should be unique, got %q", resources[0].ResourceName)
	}
}

func TestMetricTagConfigurationsFromItems(t *testing.T) {
	metricTagConfiguration := datadogV2.NewMetricTagConfigurationWithDefaults()
	metricTagConfiguration.SetId("example.configured.metric")
	metric := datadogV2.NewMetricWithDefaults()
	metric.SetId("example.configured.metric.from.list")

	configurations := metricTagConfigurationsFromItems([]datadogV2.MetricsAndMetricTagConfigurations{
		datadogV2.MetricTagConfigurationAsMetricsAndMetricTagConfigurations(metricTagConfiguration),
		datadogV2.MetricAsMetricsAndMetricTagConfigurations(metric),
		{
			UnparsedObject: map[string]interface{}{
				"id":   "example.raw.configured.metric",
				"type": "manage_tags",
			},
		},
		{
			UnparsedObject: map[string]interface{}{
				"id":   "example.plain.metric",
				"type": "metrics",
			},
		},
	})

	if len(configurations) != 4 {
		t.Fatalf("configuration count = %d, want %d", len(configurations), 4)
	}
	if configurations[0].GetId() != "example.configured.metric" {
		t.Fatalf("first configuration ID = %q, want %q", configurations[0].GetId(), "example.configured.metric")
	}
	if configurations[1].GetId() != "example.configured.metric.from.list" {
		t.Fatalf("second configuration ID = %q, want %q", configurations[1].GetId(), "example.configured.metric.from.list")
	}
	if configurations[2].GetId() != "example.raw.configured.metric" {
		t.Fatalf("third configuration ID = %q, want %q", configurations[2].GetId(), "example.raw.configured.metric")
	}
	if configurations[3].GetId() != "example.plain.metric" {
		t.Fatalf("fourth configuration ID = %q, want %q", configurations[3].GetId(), "example.plain.metric")
	}
}

func TestMetricTagConfigurationsFromRawData(t *testing.T) {
	configurations := metricTagConfigurationsFromRawData([]interface{}{
		map[string]interface{}{
			"id":   "example.first.metric",
			"type": "manage_tags",
		},
		map[string]interface{}{
			"id":   "example.second.metric",
			"type": "manage_tags",
		},
		map[string]interface{}{
			"id":   "example.plain.metric",
			"type": "metrics",
		},
		map[string]interface{}{
			"id":   "example.ignored.metric",
			"type": "unknown",
		},
		map[string]interface{}{
			"type": "manage_tags",
		},
	})

	if len(configurations) != 3 {
		t.Fatalf("configuration count = %d, want %d", len(configurations), 3)
	}
	if configurations[0].GetId() != "example.first.metric" {
		t.Fatalf("first configuration ID = %q, want %q", configurations[0].GetId(), "example.first.metric")
	}
	if configurations[1].GetId() != "example.second.metric" {
		t.Fatalf("second configuration ID = %q, want %q", configurations[1].GetId(), "example.second.metric")
	}
	if configurations[2].GetId() != "example.plain.metric" {
		t.Fatalf("third configuration ID = %q, want %q", configurations[2].GetId(), "example.plain.metric")
	}
}
