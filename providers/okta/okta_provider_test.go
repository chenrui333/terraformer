// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"strings"
	"testing"

	oktasdk "github.com/okta/okta-sdk-golang/v6/okta"
)

func TestOktaProviderInitClearsStateOnMissingAPIToken(t *testing.T) {
	provider := &OktaProvider{}
	t.Setenv("OKTA_ORG_NAME", "org")
	t.Setenv("OKTA_BASE_URL", "okta.com")
	t.Setenv("OKTA_API_TOKEN", "api-token")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.orgName != "org" || provider.baseURL != "okta.com" || provider.apiToken != "api-token" {
		t.Fatalf("expected provider state to be initialized, got orgName=%q baseURL=%q apiToken=%q", provider.orgName, provider.baseURL, provider.apiToken)
	}

	t.Setenv("OKTA_API_TOKEN", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without OKTA_API_TOKEN")
	}
	if provider.orgName != "" || provider.baseURL != "" || provider.apiToken != "" {
		t.Fatalf("expected stale provider state to be cleared, got orgName=%q baseURL=%q apiToken=%q", provider.orgName, provider.baseURL, provider.apiToken)
	}
}

func TestGetUserTypeSchemaID(t *testing.T) {
	userType := testOktaUserType(map[string]interface{}{
		"schema": map[string]interface{}{
			"href": "https://example.okta.com/api/v1/meta/schemas/user/custom",
		},
	})

	schemaID, err := getUserTypeSchemaID(userType)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if schemaID != "custom" {
		t.Fatalf("schemaID = %q, want custom", schemaID)
	}
}

func TestGetUserTypeName(t *testing.T) {
	tests := []struct {
		name     string
		userType oktasdk.UserType
		want     string
	}{
		{
			name:     "name metadata",
			userType: testOktaUserTypeWithProperties(map[string]interface{}{"name": "Custom User"}),
			want:     "Custom User",
		},
		{
			name:     "display name metadata",
			userType: testOktaUserTypeWithProperties(map[string]interface{}{"displayName": "Display User"}),
			want:     "Display User",
		},
		{
			name:     "fallback id",
			userType: testOktaUserTypeWithProperties(nil),
			want:     "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getUserTypeName(tt.userType); got != tt.want {
				t.Fatalf("getUserTypeName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetUserTypeSchemaIDAllowsMissingLinkFields(t *testing.T) {
	tests := []struct {
		name     string
		userType oktasdk.UserType
	}{
		{
			name:     "links missing",
			userType: testOktaUserTypeWithProperties(nil),
		},
		{
			name:     "schema missing",
			userType: testOktaUserType(map[string]interface{}{}),
		},
		{
			name: "href missing",
			userType: testOktaUserType(map[string]interface{}{
				"schema": map[string]interface{}{},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaID, err := getUserTypeSchemaID(tt.userType)
			if err != nil {
				t.Fatalf("expected no error: %v", err)
			}
			if schemaID != "" {
				t.Fatalf("schemaID = %q, want empty", schemaID)
			}
		})
	}
}

func TestGetUserTypeSchemaIDRejectsMalformedLinkFields(t *testing.T) {
	tests := []struct {
		name     string
		userType oktasdk.UserType
		wantErr  string
	}{
		{
			name:     "links wrong type",
			userType: testOktaUserTypeWithProperties(map[string]interface{}{"_links": "bad-links"}),
			wantErr:  "links has type",
		},
		{
			name:     "schema wrong type",
			userType: testOktaUserType(map[string]interface{}{"schema": "bad-schema"}),
			wantErr:  "schema has type",
		},
		{
			name: "href wrong type",
			userType: testOktaUserType(map[string]interface{}{
				"schema": map[string]interface{}{
					"href": 123,
				},
			}),
			wantErr: "href has type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getUserTypeSchemaID(tt.userType)
			if err == nil {
				t.Fatal("expected malformed schema link error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetUserTypeSchemaIDReturnsParseError(t *testing.T) {
	userType := testOktaUserType(map[string]interface{}{
		"schema": map[string]interface{}{
			"href": "https://example.okta.com/%zz",
		},
	})

	_, err := getUserTypeSchemaID(userType)
	if err == nil {
		t.Fatal("expected schema link parse error")
	}
	if !strings.Contains(err.Error(), "schema link") {
		t.Fatalf("error = %q, want schema link context", err)
	}
}

func TestGetUserTypeSchemaIDRejectsUnexpectedPath(t *testing.T) {
	userType := testOktaUserType(map[string]interface{}{
		"schema": map[string]interface{}{
			"href": "https://example.okta.com/api/v1/users/custom",
		},
	})

	_, err := getUserTypeSchemaID(userType)
	if err == nil {
		t.Fatal("expected unexpected schema path error")
	}
	if !strings.Contains(err.Error(), "unexpected path") {
		t.Fatalf("error = %q, want unexpected path context", err)
	}
}

func TestGetUserTypeSchemaIDRejectsMissingSchemaID(t *testing.T) {
	userType := testOktaUserType(map[string]interface{}{
		"schema": map[string]interface{}{
			"href": "https://example.okta.com/api/v1/meta/schemas/user/",
		},
	})

	_, err := getUserTypeSchemaID(userType)
	if err == nil {
		t.Fatal("expected missing schema ID error")
	}
	if !strings.Contains(err.Error(), "missing schema ID") {
		t.Fatalf("error = %q, want missing schema ID context", err)
	}
}

func testOktaUserType(links map[string]interface{}) oktasdk.UserType {
	props := map[string]interface{}{}
	if links != nil {
		props["_links"] = links
	}
	return testOktaUserTypeWithProperties(props)
}

func testOktaUserTypeWithProperties(props map[string]interface{}) oktasdk.UserType {
	if props == nil {
		props = map[string]interface{}{}
	}
	return oktasdk.UserType{
		Id:                   stringPtr("custom"),
		AdditionalProperties: props,
	}
}

func stringPtr(value string) *string {
	return &value
}
