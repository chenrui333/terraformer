// SPDX-License-Identifier: Apache-2.0

package fastly

import "testing"

func TestFastlyProviderInitClearsStateOnMissingCustomerID(t *testing.T) {
	provider := &FastlyProvider{}
	t.Setenv("FASTLY_API_KEY", "api-key")
	t.Setenv("FASTLY_CUSTOMER_ID", "customer-id")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.apiKey != "api-key" || provider.customerID != "customer-id" {
		t.Fatalf("expected provider state to be initialized, got apiKey=%q customerID=%q", provider.apiKey, provider.customerID)
	}

	t.Setenv("FASTLY_CUSTOMER_ID", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without FASTLY_CUSTOMER_ID")
	}
	if provider.apiKey != "" || provider.customerID != "" {
		t.Fatalf("expected stale provider state to be cleared, got apiKey=%q customerID=%q", provider.apiKey, provider.customerID)
	}
}
