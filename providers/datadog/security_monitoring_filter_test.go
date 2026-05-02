// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"regexp"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/zclconf/go-cty/cty"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSecurityMonitoringFilterAllowEmptyValuesPreservesQueries(t *testing.T) {
	allowEmptyValues := []*regexp.Regexp{}
	for _, pattern := range SecurityMonitoringFilterAllowEmptyValues {
		allowEmptyValues = append(allowEmptyValues, regexp.MustCompile(pattern))
	}

	parser := terraformutils.NewFlatmapParser(map[string]string{
		"query":                    "",
		"exclusion_filter.#":       "1",
		"exclusion_filter.0.name":  "catch-all",
		"exclusion_filter.0.query": "",
	}, nil, allowEmptyValues)
	filterType := cty.Object(map[string]cty.Type{
		"query": cty.String,
		"exclusion_filter": cty.List(cty.Object(map[string]cty.Type{
			"name":  cty.String,
			"query": cty.String,
		})),
	})

	result, err := parser.Parse(filterType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result["query"] != "" {
		t.Fatalf("query = %v, want empty string", result["query"])
	}
	exclusionFilters, ok := result["exclusion_filter"].([]interface{})
	if !ok {
		t.Fatalf("exclusion_filter = %T, want []interface{}", result["exclusion_filter"])
	}
	if len(exclusionFilters) != 1 {
		t.Fatalf("exclusion_filter length = %d, want %d", len(exclusionFilters), 1)
	}
	exclusionFilter, ok := exclusionFilters[0].(map[string]interface{})
	if !ok {
		t.Fatalf("exclusion_filter[0] = %T, want map[string]interface{}", exclusionFilters[0])
	}
	if exclusionFilter["query"] != "" {
		t.Fatalf("exclusion_filter[0].query = %v, want empty string", exclusionFilter["query"])
	}
}

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
