// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestIntegrationAWSAccountCreateResource(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		awsAcctID string
		wantID    string
		wantName  string
		wantType  string
	}{
		{
			name:      "uses AWS account ID in name",
			id:        "cfg-abc123",
			awsAcctID: "123456789012",
			wantID:    "cfg-abc123",
			wantName:  "tfer--integration_aws_account_123456789012",
			wantType:  "datadog_integration_aws_account",
		},
		{
			name:      "falls back to config ID",
			id:        "cfg-abc123",
			awsAcctID: "",
			wantID:    "cfg-abc123",
			wantName:  "tfer--integration_aws_account_cfg-abc123",
			wantType:  "datadog_integration_aws_account",
		},
	}

	generator := IntegrationAWSAccountGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := datadogV2.NewAWSAccountResponseDataWithDefaults()
			account.SetId(tt.id)
			if tt.awsAcctID != "" {
				attrs := datadogV2.NewAWSAccountResponseAttributesWithDefaults()
				attrs.SetAwsAccountId(tt.awsAcctID)
				account.SetAttributes(*attrs)
			}

			resource := generator.createResource(*account)
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
		})
	}
}

func TestIntegrationAWSEventBridgeCreateResource(t *testing.T) {
	generator := IntegrationAWSEventBridgeGenerator{}

	resource := generator.createResource("datadog-alerts-us-east-1-abc123", "123456789012", "us-east-1")
	if resource.InstanceState.ID != "datadog-alerts-us-east-1-abc123" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "datadog-alerts-us-east-1-abc123")
	}
	if resource.ResourceName != "tfer--integration_aws_event_bridge_datadog-alerts-us-east-1-abc123" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--integration_aws_event_bridge_datadog-alerts-us-east-1-abc123")
	}
	if resource.InstanceInfo.Type != "datadog_integration_aws_event_bridge" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_integration_aws_event_bridge")
	}
	if resource.InstanceState.Attributes["account_id"] != "123456789012" {
		t.Fatalf("account_id = %q, want %q", resource.InstanceState.Attributes["account_id"], "123456789012")
	}
	if resource.InstanceState.Attributes["region"] != "us-east-1" {
		t.Fatalf("region = %q, want %q", resource.InstanceState.Attributes["region"], "us-east-1")
	}
	if resource.InstanceState.Attributes["event_generator_name"] != "datadog-alerts-us-east-1-abc123" {
		t.Fatalf("event_generator_name = %q, want %q", resource.InstanceState.Attributes["event_generator_name"], "datadog-alerts-us-east-1-abc123")
	}
}

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
