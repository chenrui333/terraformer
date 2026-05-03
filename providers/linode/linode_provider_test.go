// SPDX-License-Identifier: Apache-2.0

package linode

import "testing"

func TestLinodeProviderInitClearsTokenOnMissingEnv(t *testing.T) {
	provider := &LinodeProvider{}
	t.Setenv("LINODE_TOKEN", "token")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.token != "token" {
		t.Fatalf("expected token to be initialized, got %q", provider.token)
	}

	t.Setenv("LINODE_TOKEN", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without LINODE_TOKEN")
	}
	if provider.token != "" {
		t.Fatalf("expected stale token to be cleared, got %q", provider.token)
	}
}
