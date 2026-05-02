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
