// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/google/uuid"
)

func TestOrganizationSettingsCreateResource(t *testing.T) {
	tests := []struct {
		name     string
		publicID string
		orgName  string
		wantID   string
		wantName string
	}{
		{
			name:     "uses org name",
			publicID: "abc123def456",
			orgName:  "my-org",
			wantID:   "abc123def456",
			wantName: "tfer--organization_settings_my-org",
		},
		{
			name:     "falls back to public ID",
			publicID: "abc123def456",
			orgName:  "",
			wantID:   "abc123def456",
			wantName: "tfer--organization_settings_abc123def456",
		},
	}

	generator := OrganizationSettingsGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := generator.createResource(tt.publicID, tt.orgName)
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != "datadog_organization_settings" {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_organization_settings")
			}
		})
	}
}

func TestOrgConnectionCreateResource(t *testing.T) {
	generator := OrgConnectionGenerator{}

	conn := datadogV2.NewOrgConnectionWithDefaults()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	conn.SetId(id)

	resource := generator.createResource(*conn)
	wantID := "11111111-2222-3333-4444-555555555555"
	if resource.InstanceState.ID != wantID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, wantID)
	}
	if resource.InstanceInfo.Type != "datadog_org_connection" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_org_connection")
	}
}

func TestOrgGroupCreateResource(t *testing.T) {
	generator := OrgGroupGenerator{}

	group := datadogV2.NewOrgGroupDataWithDefaults()
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	group.SetId(id)

	resource := generator.createResource(*group)
	wantID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	if resource.InstanceState.ID != wantID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, wantID)
	}
	if resource.InstanceInfo.Type != "datadog_org_group" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_org_group")
	}
}

func TestOrgGroupMembershipCreateResource(t *testing.T) {
	generator := OrgGroupMembershipGenerator{}

	membership := datadogV2.NewOrgGroupMembershipDataWithDefaults()
	id := uuid.MustParse("12345678-1234-1234-1234-123456789012")
	membership.SetId(id)

	resource := generator.createResource(*membership)
	wantID := "12345678-1234-1234-1234-123456789012"
	if resource.InstanceState.ID != wantID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, wantID)
	}
	if resource.InstanceInfo.Type != "datadog_org_group_membership" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_org_group_membership")
	}
}

func TestOrgGroupPolicyCreateResource(t *testing.T) {
	generator := OrgGroupPolicyGenerator{}

	policy := datadogV2.NewOrgGroupPolicyDataWithDefaults()
	id := uuid.MustParse("deadbeef-dead-beef-dead-beefdeadbeef")
	policy.SetId(id)

	orgGroupID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	resource := generator.createResource(*policy, orgGroupID)
	wantID := "deadbeef-dead-beef-dead-beefdeadbeef"
	if resource.InstanceState.ID != wantID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, wantID)
	}
	if resource.InstanceInfo.Type != "datadog_org_group_policy" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_org_group_policy")
	}
	if resource.InstanceState.Attributes["org_group_id"] != orgGroupID {
		t.Fatalf("org_group_id = %q, want %q", resource.InstanceState.Attributes["org_group_id"], orgGroupID)
	}
}

func TestOrgConnectionSkipsNilUUID(t *testing.T) {
	conn := datadogV2.NewOrgConnectionWithDefaults()
	id := conn.GetId().String()
	if id != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("expected nil UUID default, got %q", id)
	}
}
