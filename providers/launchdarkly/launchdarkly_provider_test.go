// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import "testing"

func TestLaunchDarklyProviderInitClearsStateOnMissingToken(t *testing.T) {
	provider := &LaunchDarklyProvider{}
	t.Setenv("LAUNCHDARKLY_ACCESS_TOKEN", "access-token")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.apiKey != "access-token" || provider.client == nil || provider.ctx == nil {
		t.Fatalf("expected provider state to be initialized, got apiKey=%q client=%v ctx=%v", provider.apiKey, provider.client, provider.ctx)
	}

	t.Setenv("LAUNCHDARKLY_ACCESS_TOKEN", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without LAUNCHDARKLY_ACCESS_TOKEN")
	}
	if provider.apiKey != "" || provider.client != nil || provider.ctx != nil {
		t.Fatalf("expected stale provider state to be cleared, got apiKey=%q client=%v ctx=%v", provider.apiKey, provider.client, provider.ctx)
	}
}
