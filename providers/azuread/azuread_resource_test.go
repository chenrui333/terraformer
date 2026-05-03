// SPDX-License-Identifier: Apache-2.0

package azuread

import (
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/manicminer/hamilton/msgraph"
)

func stringPtr(value string) *string {
	return &value
}

func TestAzureADAppendResourceFallsBackToIDName(t *testing.T) {
	tests := []struct {
		name     string
		append   func() ([]terraformutils.Resource, error)
		wantID   string
		wantName string
		wantType string
	}{
		{
			name: "application",
			append: func() ([]terraformutils.Resource, error) {
				generator := &ApplicationServiceGenerator{}
				err := generator.appendResource(&msgraph.Application{
					DirectoryObject: msgraph.DirectoryObject{Id: stringPtr("application-id")},
				})
				return generator.Resources, err
			},
			wantID:   "application-id",
			wantName: "application-id",
			wantType: "azuread_application",
		},
		{
			name: "group",
			append: func() ([]terraformutils.Resource, error) {
				generator := &GroupServiceGenerator{}
				err := generator.appendResource(&msgraph.Group{
					DirectoryObject: msgraph.DirectoryObject{Id: stringPtr("group-id")},
				})
				return generator.Resources, err
			},
			wantID:   "group-id",
			wantName: "group-id",
			wantType: "azuread_group",
		},
		{
			name: "service principal",
			append: func() ([]terraformutils.Resource, error) {
				generator := &ServicePrincipalServiceGenerator{}
				err := generator.appendResource(&msgraph.ServicePrincipal{
					DirectoryObject: msgraph.DirectoryObject{Id: stringPtr("principal-id")},
				})
				return generator.Resources, err
			},
			wantID:   "principal-id",
			wantName: "principal-id",
			wantType: "azuread_service_principal",
		},
		{
			name: "user",
			append: func() ([]terraformutils.Resource, error) {
				generator := &UserServiceGenerator{}
				err := generator.appendResource(&msgraph.User{
					DirectoryObject: msgraph.DirectoryObject{Id: stringPtr("user-id")},
				})
				return generator.Resources, err
			},
			wantID:   "user-id",
			wantName: "user-id",
			wantType: "azuread_user",
		},
		{
			name: "app role assignment",
			append: func() ([]terraformutils.Resource, error) {
				generator := &AppRoleAssignmentServiceGenerator{}
				err := generator.appendResource(&msgraph.AppRoleAssignment{
					Id:          stringPtr("assignment-id"),
					PrincipalId: stringPtr("principal-id"),
				})
				return generator.Resources, err
			},
			wantID:   "principal-id/appRoleAssignment/assignment-id",
			wantName: "principal-id/appRoleAssignment/assignment-id",
			wantType: "azuread_app_role_assignment",
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
			if resource.ResourceName != terraformutils.TfSanitize(tt.wantName) {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, terraformutils.TfSanitize(tt.wantName))
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
		})
	}
}

func TestAzureADAppendResourceRequiresImportIDs(t *testing.T) {
	tests := []struct {
		name    string
		append  func() error
		wantErr string
	}{
		{
			name: "application nil resource",
			append: func() error {
				return (&ApplicationServiceGenerator{}).appendResource(nil)
			},
			wantErr: "resource is nil",
		},
		{
			name: "application id",
			append: func() error {
				return (&ApplicationServiceGenerator{}).appendResource(&msgraph.Application{})
			},
			wantErr: "missing id",
		},
		{
			name: "group id",
			append: func() error {
				return (&GroupServiceGenerator{}).appendResource(&msgraph.Group{})
			},
			wantErr: "missing id",
		},
		{
			name: "service principal id",
			append: func() error {
				return (&ServicePrincipalServiceGenerator{}).appendResource(&msgraph.ServicePrincipal{})
			},
			wantErr: "missing id",
		},
		{
			name: "user id",
			append: func() error {
				return (&UserServiceGenerator{}).appendResource(&msgraph.User{})
			},
			wantErr: "missing id",
		},
		{
			name: "app role principal id",
			append: func() error {
				return (&AppRoleAssignmentServiceGenerator{}).appendResource(&msgraph.AppRoleAssignment{Id: stringPtr("assignment-id")})
			},
			wantErr: "missing principalId",
		},
		{
			name: "app role assignment id",
			append: func() error {
				return (&AppRoleAssignmentServiceGenerator{}).appendResource(&msgraph.AppRoleAssignment{PrincipalId: stringPtr("principal-id")})
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
