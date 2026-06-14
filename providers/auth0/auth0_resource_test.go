// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"strings"
	"testing"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

func auth0StringPtr(value string) *string {
	return &value
}

func auth0ActionTriggerTypePtr(value management.ActionTriggerTypeEnum) *management.ActionTriggerTypeEnum {
	return &value
}

func TestAuth0CreateResourcesFallsBackToIDName(t *testing.T) {
	tests := []struct {
		name     string
		create   func() ([]terraformutils.Resource, error)
		wantID   string
		wantName string
		wantType string
	}{
		{
			name: "action",
			create: func() ([]terraformutils.Resource, error) {
				return (ActionGenerator{}).createResources([]*management.Action{{ID: auth0StringPtr("action-id")}})
			},
			wantID:   "action-id",
			wantName: "action-id",
			wantType: "auth0_action",
		},
		{
			name: "client",
			create: func() ([]terraformutils.Resource, error) {
				return (ClientGenerator{}).createResources([]*management.Client{{ClientID: auth0StringPtr("client-id")}})
			},
			wantID:   "client-id",
			wantName: "client-id",
			wantType: "auth0_client",
		},
		{
			name: "client grant",
			create: func() ([]terraformutils.Resource, error) {
				return (ClientGrantGenerator{}).createResources([]*management.ClientGrantResponseContent{{ID: auth0StringPtr("grant-id")}})
			},
			wantID:   "grant-id",
			wantName: "grant-id",
			wantType: "auth0_client_grant",
		},
		{
			name: "custom domain",
			create: func() ([]terraformutils.Resource, error) {
				return (CustomDomainGenerator{}).createResources([]*management.CustomDomain{{CustomDomainID: "domain-id"}})
			},
			wantID:   "domain-id",
			wantName: "domain-id",
			wantType: "auth0_custom_domain",
		},
		{
			name: "email",
			create: func() ([]terraformutils.Resource, error) {
				return (EmailGenerator{}).createResources(&management.GetEmailProviderResponseContent{Name: auth0StringPtr("smtp")})
			},
			wantID:   "smtp",
			wantName: "smtp",
			wantType: "auth0_email_provider",
		},
		{
			name: "hook",
			create: func() ([]terraformutils.Resource, error) {
				return (HookGenerator{}).createResources([]*management.Hook{{ID: auth0StringPtr("hook-id")}})
			},
			wantID:   "hook-id",
			wantName: "hook-id",
			wantType: "auth0_hook",
		},
		{
			name: "log stream",
			create: func() ([]terraformutils.Resource, error) {
				return (LogStreamGenerator{}).createResources([]*management.LogStreamResponseSchema{{
					LogStreamHTTPResponseSchema: &management.LogStreamHTTPResponseSchema{ID: auth0StringPtr("stream-id")},
				}})
			},
			wantID:   "stream-id",
			wantName: "stream-id",
			wantType: "auth0_log_stream",
		},
		{
			name: "resource server",
			create: func() ([]terraformutils.Resource, error) {
				return (ResourceServerGenerator{}).createResources([]*management.ResourceServer{{ID: auth0StringPtr("server-id")}})
			},
			wantID:   "server-id",
			wantName: "server-id",
			wantType: "auth0_resource_server",
		},
		{
			name: "role",
			create: func() ([]terraformutils.Resource, error) {
				return (RoleGenerator{}).createResources([]*management.Role{{ID: auth0StringPtr("role-id")}})
			},
			wantID:   "role-id",
			wantName: "role-id",
			wantType: "auth0_role",
		},
		{
			name: "rule",
			create: func() ([]terraformutils.Resource, error) {
				return (RuleGenerator{}).createResources([]*management.Rule{{ID: auth0StringPtr("rule-id")}})
			},
			wantID:   "rule-id",
			wantName: "rule-id",
			wantType: "auth0_rule",
		},
		{
			name: "rule config",
			create: func() ([]terraformutils.Resource, error) {
				return (RuleConfigGenerator{}).createResources([]*management.RulesConfig{{Key: auth0StringPtr("config-key")}})
			},
			wantID:   "config-key",
			wantName: "config-key",
			wantType: "auth0_rule_config",
		},
		{
			name: "trigger binding",
			create: func() ([]terraformutils.Resource, error) {
				return (TriggerBindingGenerator{}).createResources(map[string][]*management.ActionBinding{
					"post-login": {
						{
							ID:          auth0StringPtr("binding-id"),
							TriggerID:   auth0ActionTriggerTypePtr("post-login"),
							Action:      &management.Action{ID: auth0StringPtr("action-id")},
							DisplayName: auth0StringPtr("Action Name"),
						},
					},
				})
			},
			wantID:   "post-login",
			wantName: "post-login",
			wantType: "auth0_trigger_actions",
		},
		{
			name: "user",
			create: func() ([]terraformutils.Resource, error) {
				return (UserGenerator{}).createResources([]*management.UserResponseSchema{{UserID: auth0StringPtr("auth0|user-id")}})
			},
			wantID:   "auth0|user-id",
			wantName: "auth0|user-id",
			wantType: "auth0_user",
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
			if resource.ResourceName != terraformutils.TfSanitize(tt.wantName) {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, terraformutils.TfSanitize(tt.wantName))
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
		})
	}
}

func TestAuth0CreateResourcesIncludesDisplayNameWhenAvailable(t *testing.T) {
	resources, err := (ActionGenerator{}).createResources([]*management.Action{{
		ID:   auth0StringPtr("action-id"),
		Name: auth0StringPtr("display-name"),
	}})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(resources))
	}
	if resources[0].ResourceName != terraformutils.TfSanitize("action-id_display-name") {
		t.Fatalf("resource name = %q, want id/display name", resources[0].ResourceName)
	}
}

func TestTriggerActionsCreateResourcesSeedsActions(t *testing.T) {
	resources, err := (TriggerBindingGenerator{}).createResources(map[string][]*management.ActionBinding{
		"post-login": {
			{
				Action:      &management.Action{ID: auth0StringPtr("action-id")},
				DisplayName: auth0StringPtr("Action Name"),
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(resources))
	}
	if resources[0].AdditionalFields["trigger"] != "post-login" {
		t.Fatalf("trigger field = %#v, want post-login", resources[0].AdditionalFields["trigger"])
	}
	actions, ok := resources[0].AdditionalFields["actions"].([]map[string]interface{})
	if !ok {
		t.Fatalf("actions field = %#v, want []map[string]interface{}", resources[0].AdditionalFields["actions"])
	}
	if len(actions) != 1 {
		t.Fatalf("actions len = %d, want 1", len(actions))
	}
	if actions[0]["id"] != "action-id" {
		t.Fatalf("action id = %#v, want action-id", actions[0]["id"])
	}
	if actions[0]["display_name"] != "Action Name" {
		t.Fatalf("action display_name = %#v, want Action Name", actions[0]["display_name"])
	}
}

func TestAuth0CreateResourcesRequiresIDs(t *testing.T) {
	tests := []struct {
		name    string
		create  func() ([]terraformutils.Resource, error)
		wantErr string
	}{
		{
			name: "action nil resource",
			create: func() ([]terraformutils.Resource, error) {
				return (ActionGenerator{}).createResources([]*management.Action{nil})
			},
			wantErr: "resource is nil",
		},
		{
			name: "action id",
			create: func() ([]terraformutils.Resource, error) {
				return (ActionGenerator{}).createResources([]*management.Action{{}})
			},
			wantErr: "missing id",
		},
		{
			name: "client id",
			create: func() ([]terraformutils.Resource, error) {
				return (ClientGenerator{}).createResources([]*management.Client{{}})
			},
			wantErr: "missing client_id",
		},
		{
			name: "client grant id",
			create: func() ([]terraformutils.Resource, error) {
				return (ClientGrantGenerator{}).createResources([]*management.ClientGrantResponseContent{{}})
			},
			wantErr: "missing id",
		},
		{
			name: "custom domain id",
			create: func() ([]terraformutils.Resource, error) {
				return (CustomDomainGenerator{}).createResources([]*management.CustomDomain{{}})
			},
			wantErr: "missing id",
		},
		{
			name: "email name",
			create: func() ([]terraformutils.Resource, error) {
				return (EmailGenerator{}).createResources(&management.GetEmailProviderResponseContent{})
			},
			wantErr: "missing name",
		},
		{
			name: "rule config nil resource",
			create: func() ([]terraformutils.Resource, error) {
				return (RuleConfigGenerator{}).createResources([]*management.RulesConfig{nil})
			},
			wantErr: "resource is nil",
		},
		{
			name: "rule config key",
			create: func() ([]terraformutils.Resource, error) {
				return (RuleConfigGenerator{}).createResources([]*management.RulesConfig{{}})
			},
			wantErr: "missing key",
		},
		{
			name: "hook id",
			create: func() ([]terraformutils.Resource, error) {
				return (HookGenerator{}).createResources([]*management.Hook{{}})
			},
			wantErr: "missing id",
		},
		{
			name: "log stream id",
			create: func() ([]terraformutils.Resource, error) {
				return (LogStreamGenerator{}).createResources([]*management.LogStreamResponseSchema{{}})
			},
			wantErr: "missing id",
		},
		{
			name: "resource server id",
			create: func() ([]terraformutils.Resource, error) {
				return (ResourceServerGenerator{}).createResources([]*management.ResourceServer{{}})
			},
			wantErr: "missing id",
		},
		{
			name: "role id",
			create: func() ([]terraformutils.Resource, error) {
				return (RoleGenerator{}).createResources([]*management.Role{{}})
			},
			wantErr: "missing id",
		},
		{
			name: "rule id",
			create: func() ([]terraformutils.Resource, error) {
				return (RuleGenerator{}).createResources([]*management.Rule{{}})
			},
			wantErr: "missing id",
		},
		{
			name: "trigger action nil binding",
			create: func() ([]terraformutils.Resource, error) {
				return (TriggerBindingGenerator{}).createResources(map[string][]*management.ActionBinding{
					"post-login": {nil},
				})
			},
			wantErr: "resource is nil",
		},
		{
			name: "trigger action action id",
			create: func() ([]terraformutils.Resource, error) {
				return (TriggerBindingGenerator{}).createResources(map[string][]*management.ActionBinding{
					"post-login": {{ID: auth0StringPtr("binding-id")}},
				})
			},
			wantErr: "missing action_id",
		},
		{
			name: "user nil resource",
			create: func() ([]terraformutils.Resource, error) {
				return (UserGenerator{}).createResources([]*management.UserResponseSchema{nil})
			},
			wantErr: "resource is nil",
		},
		{
			name: "user id",
			create: func() ([]terraformutils.Resource, error) {
				return (UserGenerator{}).createResources([]*management.UserResponseSchema{{}})
			},
			wantErr: "missing user_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.create()
			if err == nil {
				t.Fatal("expected missing Auth0 field error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}
