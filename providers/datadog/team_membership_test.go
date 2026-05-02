// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestTeamMembershipCreateResource(t *testing.T) {
	tests := []struct {
		name           string
		teamID         string
		teamMembership datadogV2.UserTeam
		wantID         string
		wantTeamID     string
		wantUserID     string
		wantName       string
		wantType       string
	}{
		{
			name:   "uses team and user relationship ids",
			teamID: "fallback-team-id",
			teamMembership: datadogV2.UserTeam{
				Id: "membership-id",
				Relationships: &datadogV2.UserTeamRelationships{
					Team: &datadogV2.RelationshipToUserTeamTeam{
						Data: datadogV2.RelationshipToUserTeamTeamData{
							Id: "team-id",
						},
					},
					User: &datadogV2.RelationshipToUserTeamUser{
						Data: datadogV2.RelationshipToUserTeamUserData{
							Id: "user-id",
						},
					},
				},
			},
			wantID:     "team-id:user-id",
			wantTeamID: "team-id",
			wantUserID: "user-id",
			wantName:   "tfer--team_membership_team-id_user-id",
			wantType:   "datadog_team_membership",
		},
		{
			name:   "falls back to supplied team id",
			teamID: "team-id",
			teamMembership: datadogV2.UserTeam{
				Id: "membership-id",
				Relationships: &datadogV2.UserTeamRelationships{
					User: &datadogV2.RelationshipToUserTeamUser{
						Data: datadogV2.RelationshipToUserTeamUserData{
							Id: "user-id",
						},
					},
				},
			},
			wantID:     "team-id:user-id",
			wantTeamID: "team-id",
			wantUserID: "user-id",
			wantName:   "tfer--team_membership_team-id_user-id",
			wantType:   "datadog_team_membership",
		},
	}

	generator := TeamMembershipGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := generator.createResource(tt.teamID, tt.teamMembership)
			if err != nil {
				t.Fatalf("createResource() error = %v", err)
			}
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.InstanceState.Attributes["team_id"] != tt.wantTeamID {
				t.Fatalf("team_id attribute = %q, want %q", resource.InstanceState.Attributes["team_id"], tt.wantTeamID)
			}
			if resource.InstanceState.Attributes["user_id"] != tt.wantUserID {
				t.Fatalf("user_id attribute = %q, want %q", resource.InstanceState.Attributes["user_id"], tt.wantUserID)
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

func TestTeamMembershipCreateResourceMissingUser(t *testing.T) {
	generator := TeamMembershipGenerator{}
	_, err := generator.createResource("team-id", datadogV2.UserTeam{Id: "membership-id"})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamMembershipCreateResourceMissingTeam(t *testing.T) {
	generator := TeamMembershipGenerator{}
	_, err := generator.createResource("", datadogV2.UserTeam{
		Id: "membership-id",
		Relationships: &datadogV2.UserTeamRelationships{
			User: &datadogV2.RelationshipToUserTeamUser{
				Data: datadogV2.RelationshipToUserTeamUserData{
					Id: "user-id",
				},
			},
		},
	})
	if err == nil {
		t.Fatal("createResource() error = nil, want error")
	}
}

func TestTeamMembershipCreateResourcesAllowsSharedUsers(t *testing.T) {
	generator := TeamMembershipGenerator{}
	membership := datadogV2.UserTeam{
		Id: "membership-id",
		Relationships: &datadogV2.UserTeamRelationships{
			User: &datadogV2.RelationshipToUserTeamUser{
				Data: datadogV2.RelationshipToUserTeamUserData{
					Id: "user-id",
				},
			},
		},
	}

	teamOneResources, err := generator.createResources("team-1", []datadogV2.UserTeam{membership})
	if err != nil {
		t.Fatalf("createResources() team-1 error = %v", err)
	}
	teamTwoResources, err := generator.createResources("team-2", []datadogV2.UserTeam{membership})
	if err != nil {
		t.Fatalf("createResources() team-2 error = %v", err)
	}
	if teamOneResources[0].ResourceName == teamTwoResources[0].ResourceName {
		t.Fatalf("resource names should be unique, got %q", teamOneResources[0].ResourceName)
	}
}

func TestParseTeamMembershipImportID(t *testing.T) {
	tests := []struct {
		name       string
		importID   string
		wantTeamID string
		wantUserID string
		wantErr    bool
	}{
		{
			name:       "valid",
			importID:   "team-id:user-id",
			wantTeamID: "team-id",
			wantUserID: "user-id",
		},
		{
			name:     "missing delimiter",
			importID: "team-id",
			wantErr:  true,
		},
		{
			name:     "missing team id",
			importID: ":user-id",
			wantErr:  true,
		},
		{
			name:     "missing user id",
			importID: "team-id:",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamID, userID, err := parseTeamMembershipImportID(tt.importID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseTeamMembershipImportID() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTeamMembershipImportID() error = %v", err)
			}
			if teamID != tt.wantTeamID {
				t.Fatalf("teamID = %q, want %q", teamID, tt.wantTeamID)
			}
			if userID != tt.wantUserID {
				t.Fatalf("userID = %q, want %q", userID, tt.wantUserID)
			}
		})
	}
}
