// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

func TestMonitorJSONCreateResource(t *testing.T) {
	generator := &MonitorJSONGenerator{}
	resource := generator.createResource("12345")

	if resource.InstanceState.ID != "12345" {
		t.Fatalf("expected resource ID 12345, got %s", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--monitor_json_12345" {
		t.Fatalf("expected resource name tfer--monitor_json_12345, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_monitor_json" {
		t.Fatalf("expected resource type datadog_monitor_json, got %s", resource.InstanceInfo.Type)
	}
}

func TestMonitorJSONCreateResourcesSkipsSyntheticsAlerts(t *testing.T) {
	metricMonitor := datadogV1.NewMonitorWithDefaults()
	metricMonitor.SetId(1001)
	metricMonitor.SetType(datadogV1.MONITORTYPE_METRIC_ALERT)

	syntheticsMonitor := datadogV1.NewMonitorWithDefaults()
	syntheticsMonitor.SetId(1002)
	syntheticsMonitor.SetType(datadogV1.MONITORTYPE_SYNTHETICS_ALERT)

	generator := &MonitorJSONGenerator{}
	resources := generator.createResources([]datadogV1.Monitor{*metricMonitor, *syntheticsMonitor})

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "1001" {
		t.Fatalf("expected resource ID 1001, got %s", resources[0].InstanceState.ID)
	}
}
