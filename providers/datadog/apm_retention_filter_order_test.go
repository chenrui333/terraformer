// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAPMRetentionFilterOrderCreateResource(t *testing.T) {
	generator := &APMRetentionFilterOrderGenerator{}
	resource := generator.createResource()

	if resource.InstanceState.ID != apmRetentionFilterOrderID {
		t.Fatalf("expected resource ID %s, got %s", apmRetentionFilterOrderID, resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--apm_retention_filter_order" {
		t.Fatalf("expected resource name tfer--apm_retention_filter_order, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_apm_retention_filter_order" {
		t.Fatalf("expected resource type datadog_apm_retention_filter_order, got %s", resource.InstanceInfo.Type)
	}
}

func TestAPMRetentionFilterOrderNormalizesIDFilter(t *testing.T) {
	generator := &APMRetentionFilterOrderGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Filter: []terraformutils.ResourceFilter{
					{
						ServiceName:      "apm_retention_filter_order",
						FieldPath:        "id",
						AcceptableValues: []string{"anything"},
					},
				},
			},
		},
	}

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if got := generator.Filter[0].AcceptableValues; len(got) != 1 || got[0] != apmRetentionFilterOrderID {
		t.Fatalf("expected normalized filter ID %s, got %v", apmRetentionFilterOrderID, got)
	}
	if !generator.Filter[0].Filter(generator.Resources[0]) {
		t.Fatal("expected normalized ID filter to keep singleton order resource")
	}
}

func TestAPMRetentionFilterOrderPostConvertHookPreservesEmptyFilterIDs(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		apmRetentionFilterOrderID,
		"apm_retention_filter_order",
		"datadog_apm_retention_filter_order",
		"datadog",
		APMRetentionFilterOrderAllowEmptyValues,
	)
	resource.InstanceState.Attributes = map[string]string{
		"filter_ids.#": "0",
	}
	resource.Item = map[string]interface{}{}

	generator := &APMRetentionFilterOrderGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	filterIDs, ok := generator.Resources[0].Item["filter_ids"].([]interface{})
	if !ok {
		t.Fatalf("filter_ids = %T, want []interface{}", generator.Resources[0].Item["filter_ids"])
	}
	if len(filterIDs) != 0 {
		t.Fatalf("filter_ids length = %d, want %d", len(filterIDs), 0)
	}
}

func TestAPMRetentionFilterOrderPostConvertHookDoesNotInventUnknownFilterIDs(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		apmRetentionFilterOrderID,
		"apm_retention_filter_order",
		"datadog_apm_retention_filter_order",
		"datadog",
		APMRetentionFilterOrderAllowEmptyValues,
	)
	resource.Item = map[string]interface{}{}

	generator := &APMRetentionFilterOrderGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}
	if _, ok := generator.Resources[0].Item["filter_ids"]; ok {
		t.Fatal("PostConvertHook added filter_ids without empty filter_ids state")
	}
}
