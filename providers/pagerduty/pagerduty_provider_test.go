// SPDX-License-Identifier: Apache-2.0

package pagerduty

import (
	"fmt"
	"strings"
	"testing"
)

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

func TestPagerDutyProviderDataOmitsToken(t *testing.T) {
	t.Setenv("PAGERDUTY_TOKEN", "pd-secret-token")
	provider := PagerDutyProvider{}
	if err := provider.Init(nil); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if rendered := fmt.Sprintf("%#v", provider.GetProviderData()); strings.Contains(rendered, "pd-secret-token") {
		t.Fatalf("provider data contains token: %s", rendered)
	}
	if got := provider.GetConfig().GetAttr("token").AsString(); got != "pd-secret-token" {
		t.Fatalf("GetConfig token = %q, want refresh credential", got)
	}
}
