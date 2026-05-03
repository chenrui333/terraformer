// SPDX-License-Identifier: Apache-2.0

package pagerduty

import "testing"

func TestPagerDutyProviderInitClearsTokenWhenEnvAndArgOmitted(t *testing.T) {
	t.Setenv("PAGERDUTY_TOKEN", "")
	provider := PagerDutyProvider{token: "old-token"}

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.token != "" {
		t.Fatalf("token = %q, want empty", provider.token)
	}
}
