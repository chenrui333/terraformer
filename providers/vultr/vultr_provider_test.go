// SPDX-License-Identifier: Apache-2.0

package vultr

import "testing"

func TestVultrProviderInitClearsAPIKeyOnMissingEnv(t *testing.T) {
	provider := &VultrProvider{}
	t.Setenv("VULTR_API_KEY", "api-key")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.apiKey != "api-key" {
		t.Fatalf("expected api key to be initialized, got %q", provider.apiKey)
	}

	t.Setenv("VULTR_API_KEY", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without VULTR_API_KEY")
	}
	if provider.apiKey != "" {
		t.Fatalf("expected stale api key to be cleared, got %q", provider.apiKey)
	}
}
