// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"strings"
	"testing"
)

func TestCommercetoolsProviderInitRequiresArgs(t *testing.T) {
	provider := CommercetoolsProvider{
		clientID:     "old-client-id",
		clientScope:  "old-scope",
		clientSecret: "old-secret",
		projectKey:   "old-project",
		baseURL:      "https://old-api.example.com",
		tokenURL:     "https://old-auth.example.com",
	}

	err := provider.Init([]string{"client-id", "scope", "secret", "project", "https://api.example.com"})
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if !strings.Contains(err.Error(), "client id, client scope, client secret, project key, base URL, and token URL are required") {
		t.Fatalf("Init error = %q, want missing Commercetools args", err)
	}
	if provider.clientID != "" || provider.clientScope != "" || provider.clientSecret != "" ||
		provider.projectKey != "" || provider.baseURL != "" || provider.tokenURL != "" {
		t.Fatalf("provider state was not cleared after failed init: %#v", provider)
	}
}

func TestCommercetoolsProviderInitStoresArgs(t *testing.T) {
	var provider CommercetoolsProvider

	err := provider.Init([]string{
		"client-id",
		"scope",
		"secret",
		"project",
		"https://api.example.com",
		"https://auth.example.com",
	})
	if err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.clientID != "client-id" {
		t.Fatalf("clientID = %q, want client-id", provider.clientID)
	}
	if provider.clientScope != "scope" {
		t.Fatalf("clientScope = %q, want scope", provider.clientScope)
	}
	if provider.clientSecret != "secret" {
		t.Fatalf("clientSecret = %q, want secret", provider.clientSecret)
	}
	if provider.projectKey != "project" {
		t.Fatalf("projectKey = %q, want project", provider.projectKey)
	}
	if provider.baseURL != "https://api.example.com" {
		t.Fatalf("baseURL = %q, want https://api.example.com", provider.baseURL)
	}
	if provider.tokenURL != "https://auth.example.com" {
		t.Fatalf("tokenURL = %q, want https://auth.example.com", provider.tokenURL)
	}
}
