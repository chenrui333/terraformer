// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestRumMetricCreateResource(t *testing.T) {
	generator := &RumMetricGenerator{}
	resource := generator.createResource("rum.sessions")

	if resource.InstanceState.ID != "rum.sessions" {
		t.Fatalf("expected resource ID rum.sessions, got %s", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--rum_metric_rum-002E-sessions" {
		t.Fatalf("expected resource name tfer--rum_metric_rum-002E-sessions, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_rum_metric" {
		t.Fatalf("expected resource type datadog_rum_metric, got %s", resource.InstanceInfo.Type)
	}
}

func TestRumMetricCreateResources(t *testing.T) {
	firstMetric := datadogV2.NewRumMetricResponseDataWithDefaults()
	firstMetric.SetId("rum.sessions")

	secondMetric := datadogV2.NewRumMetricResponseDataWithDefaults()
	secondMetric.SetId("rum.errors")

	generator := &RumMetricGenerator{}
	resources := generator.createResources([]datadogV2.RumMetricResponseData{*firstMetric, *secondMetric})

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "rum.sessions" {
		t.Fatalf("expected first resource ID rum.sessions, got %s", resources[0].InstanceState.ID)
	}
	if resources[1].InstanceState.ID != "rum.errors" {
		t.Fatalf("expected second resource ID rum.errors, got %s", resources[1].InstanceState.ID)
	}
}
