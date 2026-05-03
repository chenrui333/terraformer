// SPDX-License-Identifier: Apache-2.0

package equinixmetal

import "testing"

func TestEquinixMetalProviderInitClearsStateOnMissingProjectID(t *testing.T) {
	provider := &EquinixMetalProvider{}
	t.Setenv("PACKET_AUTH_TOKEN", "auth-token")
	t.Setenv("METAL_PROJECT_ID", "project-id")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.authToken != "auth-token" || provider.projectID != "project-id" {
		t.Fatalf("expected provider state to be initialized, got authToken=%q projectID=%q", provider.authToken, provider.projectID)
	}

	t.Setenv("METAL_PROJECT_ID", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without METAL_PROJECT_ID")
	}
	if provider.authToken != "" || provider.projectID != "" {
		t.Fatalf("expected stale provider state to be cleared, got authToken=%q projectID=%q", provider.authToken, provider.projectID)
	}
}
