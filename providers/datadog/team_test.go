// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"slices"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestTeamAllowEmptyValuesIncludesDescription(t *testing.T) {
	if !slices.Contains(TeamAllowEmptyValues, "description") {
		t.Fatal("TeamAllowEmptyValues must include description")
	}
}

func TestTeamPostConvertHookCoercesMissingDescription(t *testing.T) {
	generator := TeamGenerator{}
	generator.Resources = []terraformutils.Resource{
		{
			Item: map[string]interface{}{},
		},
		{
			Item: map[string]interface{}{
				"description": nil,
			},
		},
		{
			Item: map[string]interface{}{
				"description": "owned by platform",
			},
		},
		{},
	}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	if got := generator.Resources[0].Item["description"]; got != "" {
		t.Fatalf("missing description = %v, want empty string", got)
	}
	if got := generator.Resources[1].Item["description"]; got != "" {
		t.Fatalf("nil description = %v, want empty string", got)
	}
	if got := generator.Resources[2].Item["description"]; got != "owned by platform" {
		t.Fatalf("existing description = %v, want owned by platform", got)
	}
	if got := generator.Resources[3].Item["description"]; got != "" {
		t.Fatalf("nil item description = %v, want empty string", got)
	}
}

func TestTeamCreateResource(t *testing.T) {
	tests := []struct {
		name     string
		team     datadogV2.Team
		wantID   string
		wantName string
		wantType string
	}{
		{
			name: "uses handle and id in resource name",
			team: datadogV2.Team{
				Id: "bf064c56-edb0-11ed-ae91-da7ad0900002",
				Attributes: datadogV2.TeamAttributes{
					Handle: "platform",
					Name:   "Platform",
				},
			},
			wantID:   "bf064c56-edb0-11ed-ae91-da7ad0900002",
			wantName: "tfer--team_platform_bf064c56-edb0-11ed-ae91-da7ad0900002",
			wantType: "datadog_team",
		},
		{
			name: "falls back to id in resource name",
			team: datadogV2.Team{
				Id: "bf064c56-edb0-11ed-ae91-da7ad0900002",
			},
			wantID:   "bf064c56-edb0-11ed-ae91-da7ad0900002",
			wantName: "tfer--team_bf064c56-edb0-11ed-ae91-da7ad0900002",
			wantType: "datadog_team",
		},
	}

	generator := TeamGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := generator.createResource(tt.team)
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

func TestTeamCreateResourcesAllowsSharedHandles(t *testing.T) {
	generator := TeamGenerator{}
	resources := generator.createResources([]datadogV2.Team{
		{
			Id: "bf064c56-edb0-11ed-ae91-da7ad0900002",
			Attributes: datadogV2.TeamAttributes{
				Handle: "platform",
			},
		},
		{
			Id: "cf064c56-edb0-11ed-ae91-da7ad0900003",
			Attributes: datadogV2.TeamAttributes{
				Handle: "platform",
			},
		},
	})

	if got, want := len(resources), 2; got != want {
		t.Fatalf("len(resources) = %d, want %d", got, want)
	}
	if resources[0].ResourceName == resources[1].ResourceName {
		t.Fatalf("resource names should be unique, got %q", resources[0].ResourceName)
	}
}
