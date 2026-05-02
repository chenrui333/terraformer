// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestTeamPermissionSettingCreateResource(t *testing.T) {
	action := datadogV2.TEAMPERMISSIONSETTINGSERIALIZERACTION_MANAGE_MEMBERSHIP
	value := datadogV2.TEAMPERMISSIONSETTINGVALUE_ADMINS
	teamPermissionSetting := datadogV2.TeamPermissionSetting{
		Id: "permission-id",
		Attributes: &datadogV2.TeamPermissionSettingAttributes{
			Action: &action,
			Value:  &value,
		},
	}

	resource, err := (&TeamPermissionSettingGenerator{}).createResource("team-id", teamPermissionSetting)
	if err != nil {
		t.Fatalf("createResource() error = %v", err)
	}
	if resource.InstanceState.ID != "permission-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "permission-id")
	}
	if resource.InstanceState.Attributes["team_id"] != "team-id" {
		t.Fatalf("team_id attribute = %q, want %q", resource.InstanceState.Attributes["team_id"], "team-id")
	}
	if resource.InstanceState.Attributes["action"] != "manage_membership" {
		t.Fatalf("action attribute = %q, want %q", resource.InstanceState.Attributes["action"], "manage_membership")
	}
	if resource.InstanceState.Attributes["value"] != "admins" {
		t.Fatalf("value attribute = %q, want %q", resource.InstanceState.Attributes["value"], "admins")
	}
	if resource.ResourceName != "tfer--team_permission_setting_team-id_manage_membership" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--team_permission_setting_team-id_manage_membership")
	}
	if resource.InstanceInfo.Type != "datadog_team_permission_setting" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_team_permission_setting")
	}
}

func TestTeamPermissionSettingCreateResourceMissingID(t *testing.T) {
	action := datadogV2.TEAMPERMISSIONSETTINGSERIALIZERACTION_EDIT
	value := datadogV2.TEAMPERMISSIONSETTINGVALUE_MEMBERS
	_, err := (&TeamPermissionSettingGenerator{}).createResource("team-id", datadogV2.TeamPermissionSetting{
		Attributes: &datadogV2.TeamPermissionSettingAttributes{
			Action: &action,
			Value:  &value,
		},
	})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamPermissionSettingCreateResourceMissingTeam(t *testing.T) {
	action := datadogV2.TEAMPERMISSIONSETTINGSERIALIZERACTION_EDIT
	value := datadogV2.TEAMPERMISSIONSETTINGVALUE_MEMBERS
	_, err := (&TeamPermissionSettingGenerator{}).createResource("", datadogV2.TeamPermissionSetting{
		Id: "permission-id",
		Attributes: &datadogV2.TeamPermissionSettingAttributes{
			Action: &action,
			Value:  &value,
		},
	})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamPermissionSettingCreateResourceMissingAction(t *testing.T) {
	value := datadogV2.TEAMPERMISSIONSETTINGVALUE_MEMBERS
	_, err := (&TeamPermissionSettingGenerator{}).createResource("team-id", datadogV2.TeamPermissionSetting{
		Id: "permission-id",
		Attributes: &datadogV2.TeamPermissionSettingAttributes{
			Value: &value,
		},
	})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamPermissionSettingCreateResourceMissingValue(t *testing.T) {
	action := datadogV2.TEAMPERMISSIONSETTINGSERIALIZERACTION_EDIT
	_, err := (&TeamPermissionSettingGenerator{}).createResource("team-id", datadogV2.TeamPermissionSetting{
		Id: "permission-id",
		Attributes: &datadogV2.TeamPermissionSettingAttributes{
			Action: &action,
		},
	})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamPermissionSettingCreateResourcesAllowsSharedActions(t *testing.T) {
	action := datadogV2.TEAMPERMISSIONSETTINGSERIALIZERACTION_EDIT
	value := datadogV2.TEAMPERMISSIONSETTINGVALUE_MEMBERS
	teamPermissionSetting := datadogV2.TeamPermissionSetting{
		Id: "permission-id",
		Attributes: &datadogV2.TeamPermissionSettingAttributes{
			Action: &action,
			Value:  &value,
		},
	}

	teamOneResources, err := (&TeamPermissionSettingGenerator{}).createResources("team-1", []datadogV2.TeamPermissionSetting{teamPermissionSetting})
	if err != nil {
		t.Fatalf("createResources() team-1 error = %v", err)
	}
	teamTwoResources, err := (&TeamPermissionSettingGenerator{}).createResources("team-2", []datadogV2.TeamPermissionSetting{teamPermissionSetting})
	if err != nil {
		t.Fatalf("createResources() team-2 error = %v", err)
	}
	if teamOneResources[0].ResourceName == teamTwoResources[0].ResourceName {
		t.Fatalf("resource names should be unique, got %q", teamOneResources[0].ResourceName)
	}
}

func TestParseTeamPermissionSettingImportID(t *testing.T) {
	tests := []struct {
		name       string
		importID   string
		wantTeamID string
		wantAction string
		wantErr    bool
	}{
		{
			name:       "valid",
			importID:   "team-id:manage_membership",
			wantTeamID: "team-id",
			wantAction: "manage_membership",
		},
		{
			name:     "missing delimiter",
			importID: "team-id",
			wantErr:  true,
		},
		{
			name:     "missing team id",
			importID: ":manage_membership",
			wantErr:  true,
		},
		{
			name:     "missing action",
			importID: "team-id:",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamID, action, err := parseTeamPermissionSettingImportID(tt.importID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseTeamPermissionSettingImportID() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTeamPermissionSettingImportID() error = %v", err)
			}
			if teamID != tt.wantTeamID {
				t.Fatalf("teamID = %q, want %q", teamID, tt.wantTeamID)
			}
			if action != tt.wantAction {
				t.Fatalf("action = %q, want %q", action, tt.wantAction)
			}
		})
	}
}

func TestTeamPermissionSettingNormalizeIDFilterValues(t *testing.T) {
	action := datadogV2.TEAMPERMISSIONSETTINGSERIALIZERACTION_MANAGE_MEMBERSHIP
	value := datadogV2.TEAMPERMISSIONSETTINGVALUE_ADMINS
	resource, err := (&TeamPermissionSettingGenerator{}).createResource("team-1", datadogV2.TeamPermissionSetting{
		Id: "permission-1",
		Attributes: &datadogV2.TeamPermissionSettingAttributes{
			Action: &action,
			Value:  &value,
		},
	})
	if err != nil {
		t.Fatalf("createResource() error = %v", err)
	}

	compositeFilter := terraformutils.ResourceFilter{
		ServiceName:      "team_permission_setting",
		FieldPath:        "id",
		AcceptableValues: []string{"team-1:manage_membership"},
	}
	if compositeFilter.Filter(resource) {
		t.Fatal("composite id filter should not match resource whose state ID is the permission setting ID")
	}

	normalizedFilter := terraformutils.ResourceFilter{
		ServiceName:      "team_permission_setting",
		FieldPath:        "id",
		AcceptableValues: []string{resource.InstanceState.ID},
	}
	if !normalizedFilter.Filter(resource) {
		t.Fatal("normalized id filter should keep resource whose state ID is the permission setting ID")
	}
}
