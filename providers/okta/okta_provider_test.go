// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"strings"
	"testing"

	oktasdk "github.com/okta/okta-sdk-golang/v2/okta"
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
	userType := &oktasdk.UserType{
		Id: "custom",
		Links: map[string]interface{}{
			"schema": map[string]interface{}{
				"href": "https://example.okta.com/api/v1/meta/schemas/user/custom",
			},
		},
	}

	schemaID, err := getUserTypeSchemaID(userType)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if schemaID != "custom" {
		t.Fatalf("schemaID = %q, want custom", schemaID)
	}
}

func TestGetUserTypeSchemaIDReturnsParseError(t *testing.T) {
	userType := &oktasdk.UserType{
		Id: "custom",
		Links: map[string]interface{}{
			"schema": map[string]interface{}{
				"href": "https://example.okta.com/%zz",
			},
		},
	}

	_, err := getUserTypeSchemaID(userType)
	if err == nil {
		t.Fatal("expected schema link parse error")
	}
	if !strings.Contains(err.Error(), "schema link") {
		t.Fatalf("error = %q, want schema link context", err)
	}
}
