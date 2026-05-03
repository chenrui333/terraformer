// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"strings"
	"testing"
)

func TestAliCloudProviderInitRequiresArgs(t *testing.T) {
	provider := AliCloudProvider{
		region:  "cn-hangzhou",
		profile: "old-profile",
	}

	err := provider.Init([]string{"cn-hangzhou"})
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if !strings.Contains(err.Error(), "expected 2 init args") {
		t.Fatalf("Init error = %q, want missing AliCloud args", err)
	}
	if provider.region != "" {
		t.Fatalf("region = %q, want empty after failed init", provider.region)
	}
	if provider.profile != "" {
		t.Fatalf("profile = %q, want empty after failed init", provider.profile)
	}
}
