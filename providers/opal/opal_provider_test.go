// SPDX-License-Identifier: Apache-2.0

package opal

import "testing"

func TestOpalProviderInitClearsStateOnMissingToken(t *testing.T) {
	provider := &OpalProvider{}
	t.Setenv("OPAL_AUTH_TOKEN", "token")
	t.Setenv("OPAL_BASE_URL", "https://opal.example.com")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.token != "token" || provider.baseURL != "https://opal.example.com" {
		t.Fatalf("expected provider state to be initialized, got token=%q baseURL=%q", provider.token, provider.baseURL)
	}

	t.Setenv("OPAL_AUTH_TOKEN", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without OPAL_AUTH_TOKEN")
	}
	if provider.token != "" || provider.baseURL != "" {
		t.Fatalf("expected stale provider state to be cleared, got token=%q baseURL=%q", provider.token, provider.baseURL)
	}
}
