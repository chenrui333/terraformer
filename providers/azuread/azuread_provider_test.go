// SPDX-License-Identifier: Apache-2.0

package azuread

import "testing"

func TestAzureADProviderInitClearsStateOnMissingClientSecret(t *testing.T) {
	provider := &AzureADProvider{}
	t.Setenv("ARM_TENANT_ID", "tenant")
	t.Setenv("ARM_CLIENT_ID", "client")
	t.Setenv("ARM_CLIENT_SECRET", "secret")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.tenantID != "tenant" || provider.clientID != "client" || provider.clientSecret != "secret" {
		t.Fatalf("expected provider state to be initialized, got tenantID=%q clientID=%q clientSecret=%q", provider.tenantID, provider.clientID, provider.clientSecret)
	}

	t.Setenv("ARM_CLIENT_SECRET", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without ARM_CLIENT_SECRET")
	}
	if provider.tenantID != "" || provider.clientID != "" || provider.clientSecret != "" {
		t.Fatalf("expected stale provider state to be cleared, got tenantID=%q clientID=%q clientSecret=%q", provider.tenantID, provider.clientID, provider.clientSecret)
	}
}
