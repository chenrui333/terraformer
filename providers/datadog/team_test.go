// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
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
			InstanceState: &tfcompat.InstanceState{
				Attributes: map[string]string{},
			},
		},
		{
			Item: map[string]interface{}{
				"description": nil,
			},
			InstanceState: &tfcompat.InstanceState{
				Attributes: map[string]string{
					"id": "team-with-null-description",
				},
				TypedAttributes: typedAttributes(t, map[string]interface{}{
					"description": nil,
					"id":          "team-with-null-description",
				}),
			},
		},
		{
			Item: map[string]interface{}{
				"description": "owned by platform",
			},
			InstanceState: &tfcompat.InstanceState{
				Attributes: map[string]string{
					"description": "owned by platform",
				},
				TypedAttributes: typedAttributes(t, map[string]interface{}{
					"description": "owned by platform",
				}),
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
	if got := generator.Resources[0].InstanceState.Attributes["description"]; got != "" {
		t.Fatalf("missing description state = %v, want empty string", got)
	}
	if got := generator.Resources[1].Item["description"]; got != "" {
		t.Fatalf("nil description = %v, want empty string", got)
	}
	if got := generator.Resources[1].InstanceState.Attributes["description"]; got != "" {
		t.Fatalf("nil description state = %v, want empty string", got)
	}
	if got := typedDescription(t, generator.Resources[1].InstanceState.TypedAttributes); got != "" {
		t.Fatalf("typed description = %q, want empty string", got)
	}
	if got := generator.Resources[2].Item["description"]; got != "owned by platform" {
		t.Fatalf("existing description = %v, want owned by platform", got)
	}
	if got := generator.Resources[2].InstanceState.Attributes["description"]; got != "owned by platform" {
		t.Fatalf("existing description state = %v, want owned by platform", got)
	}
	if got := typedDescription(t, generator.Resources[2].InstanceState.TypedAttributes); got != "owned by platform" {
		t.Fatalf("existing typed description = %q, want owned by platform", got)
	}
	if got := generator.Resources[3].Item["description"]; got != "" {
		t.Fatalf("nil item description = %v, want empty string", got)
	}
}

func typedAttributes(t *testing.T, attributes map[string]interface{}) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(attributes)
	if err != nil {
		t.Fatalf("Marshal typed attributes = %v", err)
	}
	return raw
}

func typedDescription(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	attributes := map[string]string{}
	if err := json.Unmarshal(raw, &attributes); err != nil {
		t.Fatalf("Unmarshal typed attributes = %v", err)
	}
	return attributes["description"]
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
