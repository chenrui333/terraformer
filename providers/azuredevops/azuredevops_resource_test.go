// SPDX-License-Identifier: Apache-2.0

package azuredevops

import (
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
)

func uuidPtr(value string) *uuid.UUID {
	parsed := uuid.MustParse(value)
	return &parsed
}

func TestAzureDevOpsAppendResourceFallsBackToIDName(t *testing.T) {
	tests := []struct {
		name     string
		append   func() ([]terraformutils.Resource, error)
		wantID   string
		wantType string
	}{
		{
			name: "project",
			append: func() ([]terraformutils.Resource, error) {
				generator := &ProjectGenerator{}
				err := generator.appendResource(&core.TeamProjectReference{
					Id: uuidPtr("11111111-1111-1111-1111-111111111111"),
				})
				return generator.Resources, err
			},
			wantID:   "11111111-1111-1111-1111-111111111111",
			wantType: "azuredevops_project",
		},
		{
			name: "git repository",
			append: func() ([]terraformutils.Resource, error) {
				generator := &GitRepositoryGenerator{}
				err := generator.appendResource(&git.GitRepository{
					Id: uuidPtr("22222222-2222-2222-2222-222222222222"),
				})
				return generator.Resources, err
			},
			wantID:   "22222222-2222-2222-2222-222222222222",
			wantType: "azuredevops_git_repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := tt.append()
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
			if resource.ResourceName != terraformutils.TfSanitize(tt.wantID) {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, terraformutils.TfSanitize(tt.wantID))
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
		})
	}
}

func TestAzureDevOpsAppendResourceRequiresIDs(t *testing.T) {
	tests := []struct {
		name    string
		append  func() error
		wantErr string
	}{
		{
			name: "project nil resource",
			append: func() error {
				return (&ProjectGenerator{}).appendResource(nil)
			},
			wantErr: "resource is nil",
		},
		{
			name: "project id",
			append: func() error {
				return (&ProjectGenerator{}).appendResource(&core.TeamProjectReference{})
			},
			wantErr: "missing id",
		},
		{
			name: "git repository nil resource",
			append: func() error {
				return (&GitRepositoryGenerator{}).appendResource(nil)
			},
			wantErr: "resource is nil",
		},
		{
			name: "git repository id",
			append: func() error {
				return (&GitRepositoryGenerator{}).appendResource(&git.GitRepository{})
			},
			wantErr: "missing id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.append()
			if err == nil {
				t.Fatal("expected missing import ID error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}
