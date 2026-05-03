// SPDX-License-Identifier: Apache-2.0

package okta

import "testing"

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
