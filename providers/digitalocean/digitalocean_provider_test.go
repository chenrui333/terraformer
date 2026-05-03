// SPDX-License-Identifier: Apache-2.0

package digitalocean

import "testing"

func TestDigitalOceanProviderInitClearsTokenOnMissingEnv(t *testing.T) {
	provider := &DigitalOceanProvider{}
	t.Setenv("DIGITALOCEAN_TOKEN", "token")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.token != "token" {
		t.Fatalf("expected token to be initialized, got %q", provider.token)
	}

	t.Setenv("DIGITALOCEAN_TOKEN", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without DIGITALOCEAN_TOKEN")
	}
	if provider.token != "" {
		t.Fatalf("expected stale token to be cleared, got %q", provider.token)
	}
}
