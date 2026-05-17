// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestIntegrationConfluentResourceCreateResource(t *testing.T) {
	generator := IntegrationConfluentResourceGenerator{}

	resource := generator.createResource("acct-123", "res-456")
	if resource.InstanceState.ID != "acct-123:res-456" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "acct-123:res-456")
	}
	if resource.ResourceName != "tfer--integration_confluent_resource_acct-123_res-456" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--integration_confluent_resource_acct-123_res-456")
	}
	if resource.InstanceInfo.Type != "datadog_integration_confluent_resource" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_integration_confluent_resource")
	}
	if resource.InstanceState.Attributes["account_id"] != "acct-123" {
		t.Fatalf("account_id = %q, want %q", resource.InstanceState.Attributes["account_id"], "acct-123")
	}
	if resource.InstanceState.Attributes["resource_id"] != "res-456" {
		t.Fatalf("resource_id = %q, want %q", resource.InstanceState.Attributes["resource_id"], "res-456")
	}
}

func TestIntegrationFastlyServiceCreateResource(t *testing.T) {
	generator := IntegrationFastlyServiceGenerator{}

	resource := generator.createResource("acct-abc", "svc-xyz")
	if resource.InstanceState.ID != "acct-abc:svc-xyz" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "acct-abc:svc-xyz")
	}
	if resource.ResourceName != "tfer--integration_fastly_service_acct-abc_svc-xyz" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--integration_fastly_service_acct-abc_svc-xyz")
	}
	if resource.InstanceInfo.Type != "datadog_integration_fastly_service" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_integration_fastly_service")
	}
	if resource.InstanceState.Attributes["account_id"] != "acct-abc" {
		t.Fatalf("account_id = %q, want %q", resource.InstanceState.Attributes["account_id"], "acct-abc")
	}
	if resource.InstanceState.Attributes["service_id"] != "svc-xyz" {
		t.Fatalf("service_id = %q, want %q", resource.InstanceState.Attributes["service_id"], "svc-xyz")
	}
}

func TestIntegrationMSTeamsTenantBasedHandleCreateResource(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		handleName string
		wantName   string
	}{
		{
			name:       "uses handle name with id suffix",
			id:         "handle-123",
			handleName: "my-channel",
			wantName:   "tfer--integration_ms_teams_tenant_based_handle_my-channel_handle-123",
		},
		{
			name:       "falls back to id",
			id:         "handle-123",
			handleName: "",
			wantName:   "tfer--integration_ms_teams_tenant_based_handle_handle-123",
		},
	}

	generator := IntegrationMSTeamsTenantBasedHandleGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle := datadogV2.NewMicrosoftTeamsTenantBasedHandleInfoResponseDataWithDefaults()
			handle.SetId(tt.id)
			if tt.handleName != "" {
				attrs := datadogV2.NewMicrosoftTeamsTenantBasedHandleInfoResponseAttributesWithDefaults()
				attrs.SetName(tt.handleName)
				handle.SetAttributes(*attrs)
			}

			resource := generator.createResource(*handle)
			if resource.InstanceState.ID != tt.id {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.id)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != "datadog_integration_ms_teams_tenant_based_handle" {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_integration_ms_teams_tenant_based_handle")
			}
		})
	}
}
