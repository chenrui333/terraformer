// SPDX-License-Identifier: Apache-2.0

package newrelic

import "testing"

func TestNewRelicProviderInitHandlesMissingRegionArg(t *testing.T) {
	t.Setenv("NEW_RELIC_API_KEY", "")
	t.Setenv("NEW_RELIC_ACCOUNT_ID", "")

	var provider NewRelicProvider
	if err := provider.Init([]string{"api-key", "123"}); err != nil {
		t.Fatalf("expected Init to succeed without region arg: %v", err)
	}
	if provider.Region != "US" {
		t.Fatalf("Region = %q, want US", provider.Region)
	}
}

func TestNewRelicProviderInitUsesEnvForEmptyArgs(t *testing.T) {
	t.Setenv("NEW_RELIC_API_KEY", "env-key")
	t.Setenv("NEW_RELIC_ACCOUNT_ID", "123")

	var provider NewRelicProvider
	if err := provider.Init([]string{"", "", "EU"}); err != nil {
		t.Fatalf("expected Init to use env values for empty args: %v", err)
	}
	if provider.APIKey != "env-key" {
		t.Fatalf("APIKey = %q, want env-key", provider.APIKey)
	}
	if provider.accountID != 123 {
		t.Fatalf("accountID = %d, want 123", provider.accountID)
	}
	if provider.Region != "EU" {
		t.Fatalf("Region = %q, want EU", provider.Region)
	}
}

func TestNewRelicProviderInitClearsStaleOptionalState(t *testing.T) {
	t.Setenv("NEW_RELIC_API_KEY", "")
	t.Setenv("NEW_RELIC_ACCOUNT_ID", "")
	provider := NewRelicProvider{
		APIKey:    "old-key",
		accountID: 123,
		Region:    "EU",
	}

	if err := provider.Init([]string{"", "", ""}); err != nil {
		t.Fatalf("expected Init to accept empty args: %v", err)
	}
	if provider.APIKey != "" {
		t.Fatalf("APIKey = %q, want empty", provider.APIKey)
	}
	if provider.accountID != 0 {
		t.Fatalf("accountID = %d, want 0", provider.accountID)
	}
	if provider.Region != "US" {
		t.Fatalf("Region = %q, want US", provider.Region)
	}
}
