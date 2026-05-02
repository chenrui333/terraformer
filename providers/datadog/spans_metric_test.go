// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestSpansMetricCreateResource(t *testing.T) {
	generator := &SpansMetricGenerator{}
	resource := generator.createResource("trace.duration")

	if resource.InstanceState.ID != "trace.duration" {
		t.Fatalf("expected resource ID trace.duration, got %s", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--spans_metric_trace-002E-duration" {
		t.Fatalf("expected resource name tfer--spans_metric_trace-002E-duration, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_spans_metric" {
		t.Fatalf("expected resource type datadog_spans_metric, got %s", resource.InstanceInfo.Type)
	}
}

func TestSpansMetricCreateResources(t *testing.T) {
	firstMetric := datadogV2.NewSpansMetricResponseDataWithDefaults()
	firstMetric.SetId("trace.duration")

	secondMetric := datadogV2.NewSpansMetricResponseDataWithDefaults()
	secondMetric.SetId("trace.errors")

	generator := &SpansMetricGenerator{}
	resources := generator.createResources([]datadogV2.SpansMetricResponseData{*firstMetric, *secondMetric})

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "trace.duration" {
		t.Fatalf("expected first resource ID trace.duration, got %s", resources[0].InstanceState.ID)
	}
	if resources[1].InstanceState.ID != "trace.errors" {
		t.Fatalf("expected second resource ID trace.errors, got %s", resources[1].InstanceState.ID)
	}
}
