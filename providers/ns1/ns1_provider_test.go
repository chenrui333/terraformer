// SPDX-License-Identifier: Apache-2.0

package ns1

import "testing"

func TestNs1ProviderInitClearsAPIKeyOnMissingEnv(t *testing.T) {
	provider := &Ns1Provider{}
	t.Setenv("NS1_APIKEY", "api-key")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.apiKey != "api-key" {
		t.Fatalf("expected api key to be initialized, got %q", provider.apiKey)
	}

	t.Setenv("NS1_APIKEY", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without NS1_APIKEY")
	}
	if provider.apiKey != "" {
		t.Fatalf("expected stale api key to be cleared, got %q", provider.apiKey)
	}
}
