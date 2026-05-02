// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestSecurityMonitoringFilterCreateResource(t *testing.T) {
	securityFilter := datadogV2.NewSecurityFilterWithDefaults()
	securityFilter.SetId("filter-id")

	generator := &SecurityMonitoringFilterGenerator{}
	resource, err := generator.createResource(*securityFilter)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "filter-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "filter-id")
	}
	if resource.ResourceName != "tfer--security_monitoring_filter_filter-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--security_monitoring_filter_filter-id")
	}
	if resource.InstanceInfo.Type != "datadog_security_monitoring_filter" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_security_monitoring_filter")
	}
}

func TestSecurityMonitoringFilterCreateResourceMissingID(t *testing.T) {
	generator := &SecurityMonitoringFilterGenerator{}
	_, err := generator.createResource(datadogV2.SecurityFilter{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestSecurityMonitoringFilterCreateResources(t *testing.T) {
	firstFilter := datadogV2.NewSecurityFilterWithDefaults()
	firstFilter.SetId("filter-1")
	secondFilter := datadogV2.NewSecurityFilterWithDefaults()
	secondFilter.SetId("filter-2")

	generator := &SecurityMonitoringFilterGenerator{}
	resources, err := generator.createResources([]datadogV2.SecurityFilter{
		*firstFilter,
		*secondFilter,
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

func TestSecurityMonitoringFiltersFromRawData(t *testing.T) {
	securityFilters := securityMonitoringFiltersFromRawData([]interface{}{
		map[string]interface{}{
			"id":   "filter-1",
			"type": "security_filters",
		},
		map[string]interface{}{
			"id": "filter-2",
		},
		map[string]interface{}{
			"id":   "ignored-type",
			"type": "unknown",
		},
		map[string]interface{}{
			"type": "security_filters",
		},
	})

	if len(securityFilters) != 2 {
		t.Fatalf("filter count = %d, want %d", len(securityFilters), 2)
	}
	if securityFilters[0].GetId() != "filter-1" {
		t.Fatalf("first filter ID = %q, want %q", securityFilters[0].GetId(), "filter-1")
	}
	if securityFilters[1].GetId() != "filter-2" {
		t.Fatalf("second filter ID = %q, want %q", securityFilters[1].GetId(), "filter-2")
	}
}

func TestSecurityMonitoringFilterFromRawDataRejectsNonObjects(t *testing.T) {
	if _, ok := securityMonitoringFilterFromRawData("filter-id"); ok {
		t.Fatal("securityMonitoringFilterFromRawData accepted non-object raw data")
	}
}
