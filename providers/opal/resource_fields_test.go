// SPDX-License-Identifier: Apache-2.0

package opal

import (
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	opalsdk "github.com/opalsecurity/opal-go"
)

func opalStringPtr(value string) *string {
	return &value
}

func TestOpalCreateResourcesFallsBackToIDName(t *testing.T) {
	tests := []struct {
		name     string
		create   func() ([]terraformutils.Resource, error)
		wantID   string
		wantName string
		wantType string
	}{
		{
			name: "group",
			create: func() ([]terraformutils.Resource, error) {
				return (&GroupGenerator{}).createResources([]opalsdk.Group{{GroupId: "group-id"}}, map[string]int{})
			},
			wantID:   "group-id",
			wantName: "group_id",
			wantType: "opal_group",
		},
		{
			name: "message channel",
			create: func() ([]terraformutils.Resource, error) {
				return (&MessageChannelGenerator{}).createResources([]opalsdk.MessageChannel{{MessageChannelId: "channel-id"}})
			},
			wantID:   "channel-id",
			wantName: "channel_id",
			wantType: "opal_message_channel",
		},
		{
			name: "on call schedule",
			create: func() ([]terraformutils.Resource, error) {
				return (&OnCallScheduleGenerator{}).createResources([]opalsdk.OnCallSchedule{{OnCallScheduleId: opalStringPtr("schedule-id")}})
			},
			wantID:   "schedule-id",
			wantName: "schedule_id",
			wantType: "opal_on_call_schedule",
		},
		{
			name: "owner",
			create: func() ([]terraformutils.Resource, error) {
				return (&OwnerGenerator{}).createResources([]opalsdk.Owner{{OwnerId: "owner-id"}}, map[string]int{})
			},
			wantID:   "owner-id",
			wantName: "owner_id",
			wantType: "opal_owner",
		},
		{
			name: "resource",
			create: func() ([]terraformutils.Resource, error) {
				return (&ResourceGenerator{}).createResources([]*opalsdk.Resource{{ResourceId: "resource-id"}})
			},
			wantID:   "resource-id",
			wantName: "resource_id",
			wantType: "opal_resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := tt.create()
			if err != nil {
				t.Fatalf("expected no error: %v", err)
			}
			if len(resources) != 1 {
				t.Fatalf("resources len = %d, want 1", len(resources))
			}
			resource := resources[0]
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			wantName := terraformutils.TfSanitize(tt.wantName)
			if resource.ResourceName != wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
		})
	}
}

func TestOpalCreateResourcesIncludesDisplayNameWhenAvailable(t *testing.T) {
	resources, err := (&GroupGenerator{}).createResources([]opalsdk.Group{{
		GroupId: "group-id",
		Name:    opalStringPtr("Engineering Team"),
	}}, map[string]int{})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(resources))
	}
	wantName := terraformutils.TfSanitize("engineering_team")
	if resources[0].ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resources[0].ResourceName, wantName)
	}
}

func TestOpalCreateResourcesRequiresIDs(t *testing.T) {
	tests := []struct {
		name    string
		create  func() ([]terraformutils.Resource, error)
		wantErr string
	}{
		{
			name: "group id",
			create: func() ([]terraformutils.Resource, error) {
				return (&GroupGenerator{}).createResources([]opalsdk.Group{{}}, map[string]int{})
			},
			wantErr: "missing group_id",
		},
		{
			name: "message channel id",
			create: func() ([]terraformutils.Resource, error) {
				return (&MessageChannelGenerator{}).createResources([]opalsdk.MessageChannel{{}})
			},
			wantErr: "missing message_channel_id",
		},
		{
			name: "on call schedule id",
			create: func() ([]terraformutils.Resource, error) {
				return (&OnCallScheduleGenerator{}).createResources([]opalsdk.OnCallSchedule{{}})
			},
			wantErr: "missing on_call_schedule_id",
		},
		{
			name: "owner id",
			create: func() ([]terraformutils.Resource, error) {
				return (&OwnerGenerator{}).createResources([]opalsdk.Owner{{}}, map[string]int{})
			},
			wantErr: "missing owner_id",
		},
		{
			name: "resource nil",
			create: func() ([]terraformutils.Resource, error) {
				return (&ResourceGenerator{}).createResources([]*opalsdk.Resource{nil})
			},
			wantErr: "resource is nil",
		},
		{
			name: "resource id",
			create: func() ([]terraformutils.Resource, error) {
				return (&ResourceGenerator{}).createResources([]*opalsdk.Resource{{}})
			},
			wantErr: "missing resource_id",
		},
		{
			name: "permission set parent resource id",
			create: func() ([]terraformutils.Resource, error) {
				resourceType := opalsdk.RESOURCETYPEENUM_AWS_SSO_PERMISSION_SET
				return (&ResourceGenerator{}).createResources([]*opalsdk.Resource{{
					ResourceId:   "permission-set-id",
					ResourceType: &resourceType,
				}})
			},
			wantErr: "missing parent_resource_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.create()
			if err == nil {
				t.Fatal("expected missing Opal field error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestOpalUniqueResourceNameDeduplicatesFallbackNames(t *testing.T) {
	resources, err := (&GroupGenerator{}).createResources([]opalsdk.Group{
		{GroupId: "first-group", Name: opalStringPtr("Engineering")},
		{GroupId: "second-group", Name: opalStringPtr("Engineering")},
	}, map[string]int{})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resources len = %d, want 2", len(resources))
	}
	wantFirstName := terraformutils.TfSanitize("engineering")
	if resources[0].ResourceName != wantFirstName {
		t.Fatalf("first resource name = %q, want %q", resources[0].ResourceName, wantFirstName)
	}
	wantSecondName := terraformutils.TfSanitize("engineering_2")
	if resources[1].ResourceName != wantSecondName {
		t.Fatalf("second resource name = %q, want %q", resources[1].ResourceName, wantSecondName)
	}
}

func TestOpalUniqueResourceNameAvoidsGeneratedNameCollision(t *testing.T) {
	resources, err := (&GroupGenerator{}).createResources([]opalsdk.Group{
		{GroupId: "first-group", Name: opalStringPtr("Engineering")},
		{GroupId: "second-group", Name: opalStringPtr("Engineering 2")},
		{GroupId: "third-group", Name: opalStringPtr("Engineering")},
	}, map[string]int{})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(resources) != 3 {
		t.Fatalf("resources len = %d, want 3", len(resources))
	}
	wantName := terraformutils.TfSanitize("engineering_3")
	if resources[2].ResourceName != wantName {
		t.Fatalf("third resource name = %q, want %q", resources[2].ResourceName, wantName)
	}
}

func TestOpalResourcePermissionSetUsesParentFallbackName(t *testing.T) {
	resourceType := opalsdk.RESOURCETYPEENUM_AWS_SSO_PERMISSION_SET
	resources, err := (&ResourceGenerator{}).createResources([]*opalsdk.Resource{
		{ResourceId: "account-id"},
		{
			ResourceId:       "permission-set-id",
			Name:             opalStringPtr("Admin Access"),
			ResourceType:     &resourceType,
			ParentResourceId: opalStringPtr("account-id"),
		},
	})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resources len = %d, want 2", len(resources))
	}
	wantName := terraformutils.TfSanitize("account_id_admin_access")
	if resources[1].ResourceName != wantName {
		t.Fatalf("permission set name = %q, want %q", resources[1].ResourceName, wantName)
	}
}

func TestOpalResourceDeduplicatesAfterNormalization(t *testing.T) {
	resources, err := (&ResourceGenerator{}).createResources([]*opalsdk.Resource{
		{ResourceId: "abc", Name: opalStringPtr("Shared Name")},
		{ResourceId: "def", Name: opalStringPtr("Shared-Name")},
	})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resources len = %d, want 2", len(resources))
	}
	wantName := terraformutils.TfSanitize("shared_name_def")
	if resources[1].ResourceName != wantName {
		t.Fatalf("second resource name = %q, want %q", resources[1].ResourceName, wantName)
	}
}

func TestOpalResourceDuplicateNameUsesShortIDSuffix(t *testing.T) {
	resources, err := (&ResourceGenerator{}).createResources([]*opalsdk.Resource{
		{ResourceId: "abc", Name: opalStringPtr("Shared")},
		{ResourceId: "def", Name: opalStringPtr("Shared")},
	})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resources len = %d, want 2", len(resources))
	}
	wantName := terraformutils.TfSanitize("shared_def")
	if resources[1].ResourceName != wantName {
		t.Fatalf("second resource name = %q, want %q", resources[1].ResourceName, wantName)
	}
}
