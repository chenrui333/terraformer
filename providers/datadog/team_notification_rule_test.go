// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestTeamNotificationRuleCreateResource(t *testing.T) {
	ruleID := "rule-id"
	teamNotificationRule := datadogV2.TeamNotificationRule{
		Id: &ruleID,
	}

	resource, err := (&TeamNotificationRuleGenerator{}).createResource("team-id", teamNotificationRule)
	if err != nil {
		t.Fatalf("createResource() error = %v", err)
	}
	if resource.InstanceState.ID != "rule-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "rule-id")
	}
	if resource.InstanceState.Attributes["team_id"] != "team-id" {
		t.Fatalf("team_id attribute = %q, want %q", resource.InstanceState.Attributes["team_id"], "team-id")
	}
	if resource.ResourceName != "tfer--team_notification_rule_team-id_rule-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--team_notification_rule_team-id_rule-id")
	}
	if resource.InstanceInfo.Type != "datadog_team_notification_rule" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_team_notification_rule")
	}
}

func TestTeamNotificationRuleCreateResourceMissingID(t *testing.T) {
	_, err := (&TeamNotificationRuleGenerator{}).createResource("team-id", datadogV2.TeamNotificationRule{})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamNotificationRuleCreateResourceMissingTeam(t *testing.T) {
	ruleID := "rule-id"
	_, err := (&TeamNotificationRuleGenerator{}).createResource("", datadogV2.TeamNotificationRule{
		Id: &ruleID,
	})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamNotificationRuleCreateResourcesAllowsSharedRuleIDs(t *testing.T) {
	ruleID := "rule-id"
	teamNotificationRule := datadogV2.TeamNotificationRule{
		Id: &ruleID,
	}

	teamOneResources, err := (&TeamNotificationRuleGenerator{}).createResources("team-1", []datadogV2.TeamNotificationRule{teamNotificationRule})
	if err != nil {
		t.Fatalf("createResources() team-1 error = %v", err)
	}
	teamTwoResources, err := (&TeamNotificationRuleGenerator{}).createResources("team-2", []datadogV2.TeamNotificationRule{teamNotificationRule})
	if err != nil {
		t.Fatalf("createResources() team-2 error = %v", err)
	}
	if teamOneResources[0].ResourceName == teamTwoResources[0].ResourceName {
		t.Fatalf("resource names should be unique, got %q", teamOneResources[0].ResourceName)
	}
}

func TestTeamNotificationRuleFromResponse(t *testing.T) {
	ruleID := "rule-id"
	teamNotificationRule := datadogV2.TeamNotificationRule{
		Id: &ruleID,
	}

	tests := []struct {
		name     string
		response datadogV2.TeamNotificationRuleResponse
		request  string
		wantID   string
		wantOK   bool
	}{
		{
			name: "parsed data",
			response: datadogV2.TeamNotificationRuleResponse{
				Data: &teamNotificationRule,
			},
			request: "requested-rule-id",
			wantID:  "rule-id",
			wantOK:  true,
		},
		{
			name: "minimal unparsed data",
			response: datadogV2.TeamNotificationRuleResponse{
				UnparsedObject: map[string]interface{}{
					"data": map[string]interface{}{
						"id":   "rule-id",
						"type": "team_notification_rules",
					},
				},
			},
			request: "requested-rule-id",
			wantID:  "rule-id",
			wantOK:  true,
		},
		{
			name: "minimal unparsed data without id",
			response: datadogV2.TeamNotificationRuleResponse{
				UnparsedObject: map[string]interface{}{
					"data": map[string]interface{}{
						"type": "team_notification_rules",
					},
				},
			},
			request: "requested-rule-id",
			wantID:  "requested-rule-id",
			wantOK:  true,
		},
		{
			name:     "no data",
			response: datadogV2.TeamNotificationRuleResponse{},
			request:  "requested-rule-id",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := teamNotificationRuleFromResponse(tt.response, tt.request)
			if ok != tt.wantOK {
				t.Fatalf("teamNotificationRuleFromResponse() ok = %t, want %t", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if got.GetId() != tt.wantID {
				t.Fatalf("rule ID = %q, want %q", got.GetId(), tt.wantID)
			}
		})
	}
}

func TestParseTeamNotificationRuleImportID(t *testing.T) {
	tests := []struct {
		name       string
		importID   string
		wantTeamID string
		wantRuleID string
		wantErr    bool
	}{
		{
			name:       "valid",
			importID:   "team-id:rule-id",
			wantTeamID: "team-id",
			wantRuleID: "rule-id",
		},
		{
			name:     "missing delimiter",
			importID: "team-id",
			wantErr:  true,
		},
		{
			name:     "missing team id",
			importID: ":rule-id",
			wantErr:  true,
		},
		{
			name:     "missing rule id",
			importID: "team-id:",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamID, ruleID, err := parseTeamNotificationRuleImportID(tt.importID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseTeamNotificationRuleImportID() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTeamNotificationRuleImportID() error = %v", err)
			}
			if teamID != tt.wantTeamID {
				t.Fatalf("teamID = %q, want %q", teamID, tt.wantTeamID)
			}
			if ruleID != tt.wantRuleID {
				t.Fatalf("ruleID = %q, want %q", ruleID, tt.wantRuleID)
			}
		})
	}
}

func TestTeamNotificationRuleNormalizeIDFilterValues(t *testing.T) {
	filterIDs, err := parseTeamNotificationRuleImportIDs([]string{"team-1:rule-1", "team-2:rule-2"})
	if err != nil {
		t.Fatalf("parseTeamNotificationRuleImportIDs() error = %v", err)
	}

	ruleID := "rule-1"
	resource, err := (&TeamNotificationRuleGenerator{}).createResource("team-1", datadogV2.TeamNotificationRule{Id: &ruleID})
	if err != nil {
		t.Fatalf("createResource() error = %v", err)
	}

	compositeFilter := terraformutils.ResourceFilter{
		ServiceName:      "team_notification_rule",
		FieldPath:        "id",
		AcceptableValues: []string{"team-1:rule-1", "team-2:rule-2"},
	}
	if compositeFilter.Filter(resource) {
		t.Fatal("composite id filter should not match resource whose state ID is the rule ID")
	}

	generator := TeamNotificationRuleGenerator{}
	generator.Filter = []terraformutils.ResourceFilter{
		{
			ServiceName:      "team_notification_rule",
			FieldPath:        "id",
			AcceptableValues: []string{"team-1:rule-1", "team-2:rule-2"},
		},
	}
	generator.Filter[0].AcceptableValues = teamNotificationRuleIDs(filterIDs)

	if !generator.Filter[0].Filter(resource) {
		t.Fatal("normalized id filter should keep resource whose state ID is the rule ID")
	}
}
