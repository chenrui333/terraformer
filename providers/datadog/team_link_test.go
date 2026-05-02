// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestTeamLinkCreateResource(t *testing.T) {
	tests := []struct {
		name     string
		teamID   string
		teamLink datadogV2.TeamLink
		wantID   string
		wantTeam string
		wantName string
		wantType string
	}{
		{
			name:   "uses team id from attributes",
			teamID: "fallback-team-id",
			teamLink: datadogV2.TeamLink{
				Id: "link-id",
				Attributes: datadogV2.TeamLinkAttributes{
					TeamId: stringPtr("team-id"),
				},
			},
			wantID:   "link-id",
			wantTeam: "team-id",
			wantName: "tfer--team_link_team-id_link-id",
			wantType: "datadog_team_link",
		},
		{
			name:   "falls back to supplied team id",
			teamID: "team-id",
			teamLink: datadogV2.TeamLink{
				Id: "link-id",
			},
			wantID:   "link-id",
			wantTeam: "team-id",
			wantName: "tfer--team_link_team-id_link-id",
			wantType: "datadog_team_link",
		},
	}

	generator := TeamLinkGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := generator.createResource(tt.teamID, tt.teamLink)
			if err != nil {
				t.Fatalf("createResource() error = %v", err)
			}
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.InstanceState.Attributes["team_id"] != tt.wantTeam {
				t.Fatalf("team_id attribute = %q, want %q", resource.InstanceState.Attributes["team_id"], tt.wantTeam)
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

func TestTeamLinkCreateResourceMissingID(t *testing.T) {
	generator := TeamLinkGenerator{}
	_, err := generator.createResource("team-id", datadogV2.TeamLink{})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamLinkCreateResourceMissingTeam(t *testing.T) {
	generator := TeamLinkGenerator{}
	_, err := generator.createResource("", datadogV2.TeamLink{Id: "link-id"})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamLinkCreateResourcesAllowsSharedLinkIDs(t *testing.T) {
	generator := TeamLinkGenerator{}
	teamLink := datadogV2.TeamLink{
		Id: "link-id",
	}

	teamOneResources, err := generator.createResources("team-1", []datadogV2.TeamLink{teamLink})
	if err != nil {
		t.Fatalf("createResources() team-1 error = %v", err)
	}
	teamTwoResources, err := generator.createResources("team-2", []datadogV2.TeamLink{teamLink})
	if err != nil {
		t.Fatalf("createResources() team-2 error = %v", err)
	}
	if teamOneResources[0].ResourceName == teamTwoResources[0].ResourceName {
		t.Fatalf("resource names should be unique, got %q", teamOneResources[0].ResourceName)
	}
}

func TestParseTeamLinkImportID(t *testing.T) {
	tests := []struct {
		name       string
		importID   string
		wantTeamID string
		wantLinkID string
		wantErr    bool
	}{
		{
			name:       "valid",
			importID:   "team-id:link-id",
			wantTeamID: "team-id",
			wantLinkID: "link-id",
		},
		{
			name:     "missing delimiter",
			importID: "team-id",
			wantErr:  true,
		},
		{
			name:     "missing team id",
			importID: ":link-id",
			wantErr:  true,
		},
		{
			name:     "missing link id",
			importID: "team-id:",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamID, linkID, err := parseTeamLinkImportID(tt.importID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseTeamLinkImportID() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTeamLinkImportID() error = %v", err)
			}
			if teamID != tt.wantTeamID {
				t.Fatalf("teamID = %q, want %q", teamID, tt.wantTeamID)
			}
			if linkID != tt.wantLinkID {
				t.Fatalf("linkID = %q, want %q", linkID, tt.wantLinkID)
			}
		})
	}
}

func TestTeamLinkNormalizeIDFilterValues(t *testing.T) {
	filterIDs, err := parseTeamLinkImportIDs([]string{"team-1:link-1", "team-2:link-2"})
	if err != nil {
		t.Fatalf("parseTeamLinkImportIDs() error = %v", err)
	}

	resource, err := (&TeamLinkGenerator{}).createResource("team-1", datadogV2.TeamLink{Id: "link-1"})
	if err != nil {
		t.Fatalf("createResource() error = %v", err)
	}

	compositeFilter := terraformutils.ResourceFilter{
		ServiceName:      "team_link",
		FieldPath:        "id",
		AcceptableValues: []string{"team-1:link-1", "team-2:link-2"},
	}
	if compositeFilter.Filter(resource) {
		t.Fatal("composite id filter should not match resource whose state ID is the link ID")
	}

	generator := TeamLinkGenerator{}
	generator.Filter = []terraformutils.ResourceFilter{
		{
			ServiceName:      "team_link",
			FieldPath:        "id",
			AcceptableValues: []string{"team-1:link-1", "team-2:link-2"},
		},
	}
	generator.Filter[0].AcceptableValues = teamLinkIDs(filterIDs)

	if !generator.Filter[0].Filter(resource) {
		t.Fatal("normalized id filter should keep resource whose state ID is the link ID")
	}
}

func stringPtr(value string) *string {
	return &value
}
