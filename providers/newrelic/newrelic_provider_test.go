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
