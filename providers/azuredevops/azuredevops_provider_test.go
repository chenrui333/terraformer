// SPDX-License-Identifier: Apache-2.0

package azuredevops

import "testing"

func TestAzureDevOpsProviderInitClearsStateOnMissingToken(t *testing.T) {
	provider := &AzureDevOpsProvider{}
	t.Setenv("AZDO_ORG_SERVICE_URL", "https://dev.azure.com/example")
	t.Setenv("AZDO_PERSONAL_ACCESS_TOKEN", "pat")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.organizationURL != "https://dev.azure.com/example" || provider.personalAccessToken != "pat" {
		t.Fatalf("expected provider state to be initialized, got organizationURL=%q personalAccessToken=%q", provider.organizationURL, provider.personalAccessToken)
	}

	t.Setenv("AZDO_PERSONAL_ACCESS_TOKEN", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without AZDO_PERSONAL_ACCESS_TOKEN")
	}
	if provider.organizationURL != "" || provider.personalAccessToken != "" {
		t.Fatalf("expected stale provider state to be cleared, got organizationURL=%q personalAccessToken=%q", provider.organizationURL, provider.personalAccessToken)
	}
}
