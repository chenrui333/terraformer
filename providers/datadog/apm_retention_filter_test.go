// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestAPMRetentionFilterCreateResource(t *testing.T) {
	generator := &APMRetentionFilterGenerator{}
	resource := generator.createResource("rf-123")

	if resource.InstanceState.ID != "rf-123" {
		t.Fatalf("expected resource ID rf-123, got %s", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--apm_retention_filter_rf-123" {
		t.Fatalf("expected resource name tfer--apm_retention_filter_rf-123, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_apm_retention_filter" {
		t.Fatalf("expected resource type datadog_apm_retention_filter, got %s", resource.InstanceInfo.Type)
	}
}

func TestAPMRetentionFilterCreateResources(t *testing.T) {
	firstFilter := datadogV2.NewRetentionFilterAllWithDefaults()
	firstFilter.SetId("rf-123")
	firstFilter.SetAttributes(importableAPMRetentionFilterAttributes())

	secondFilter := datadogV2.NewRetentionFilterAllWithDefaults()
	secondFilter.SetId("rf-456")
	secondFilter.SetAttributes(importableAPMRetentionFilterAttributes())

	generator := &APMRetentionFilterGenerator{}
	resources := generator.createResources([]datadogV2.RetentionFilterAll{*firstFilter, *secondFilter})

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "rf-123" {
		t.Fatalf("expected first resource ID rf-123, got %s", resources[0].InstanceState.ID)
	}
	if resources[1].InstanceState.ID != "rf-456" {
		t.Fatalf("expected second resource ID rf-456, got %s", resources[1].InstanceState.ID)
	}
}

func TestAPMRetentionFilterCreateResourcesSkipsUnsupportedFilterTypes(t *testing.T) {
	userFilter := datadogV2.NewRetentionFilterAllWithDefaults()
	userFilter.SetId("rf-user")
	userFilter.SetAttributes(importableAPMRetentionFilterAttributes())

	defaultFilter := datadogV2.NewRetentionFilterAllWithDefaults()
	defaultFilter.SetId("rf-default")
	attributes := importableAPMRetentionFilterAttributes()
	attributes.SetFilterType(datadogV2.RETENTIONFILTERALLTYPE_SPANS_ERRORS_SAMPLING_PROCESSOR)
	defaultFilter.SetAttributes(attributes)

	generator := &APMRetentionFilterGenerator{}
	resources := generator.createResources([]datadogV2.RetentionFilterAll{*userFilter, *defaultFilter})

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "rf-user" {
		t.Fatalf("expected resource ID rf-user, got %s", resources[0].InstanceState.ID)
	}
}

func importableAPMRetentionFilterAttributes() datadogV2.RetentionFilterAllAttributes {
	attributes := datadogV2.NewRetentionFilterAllAttributesWithDefaults()
	attributes.SetFilterType(datadogV2.RETENTIONFILTERALLTYPE_SPANS_SAMPLING_PROCESSOR)
	return *attributes
}
