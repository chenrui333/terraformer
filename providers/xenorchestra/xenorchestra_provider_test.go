// SPDX-License-Identifier: Apache-2.0

package xenorchestra

import (
	"fmt"
	"strings"
	"testing"
)

func TestXenorchestraProviderInitClearsStateOnMissingPassword(t *testing.T) {
	provider := &XenorchestraProvider{}
	t.Setenv("XOA_URL", "https://example.com")
	t.Setenv("XOA_USER", "user")
	t.Setenv("XOA_PASSWORD", "password")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.url != "https://example.com" || provider.user != "user" || provider.password != "password" {
		t.Fatalf("expected provider state to be initialized, got url=%q user=%q password=%q", provider.url, provider.user, provider.password)
	}

	t.Setenv("XOA_PASSWORD", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without XOA_PASSWORD")
	}
	if provider.url != "" || provider.user != "" || provider.password != "" {
		t.Fatalf("expected stale provider state to be cleared, got url=%q user=%q password=%q", provider.url, provider.user, provider.password)
	}
}

func TestXenorchestraProviderDataOmitsCredentials(t *testing.T) {
	t.Setenv("XOA_URL", "https://xoa.example.com")
	t.Setenv("XOA_USER", "xoa-secret-user")
	t.Setenv("XOA_PASSWORD", "xoa-secret-password")
	provider := &XenorchestraProvider{}
	if err := provider.Init(nil); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	rendered := fmt.Sprintf("%#v", provider.GetProviderData())
	for _, secret := range []string{"xoa-secret-user", "xoa-secret-password"} {
		if strings.Contains(rendered, secret) {
			t.Fatalf("provider data contains secret %q: %s", secret, rendered)
		}
	}
	if !strings.Contains(rendered, "https://xoa.example.com") {
		t.Fatalf("provider data missing url: %s", rendered)
	}
}
