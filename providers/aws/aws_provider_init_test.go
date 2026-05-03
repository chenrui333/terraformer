// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"
	"testing"
)

func TestAWSProviderInitRequiresArgs(t *testing.T) {
	provider := AWSProvider{
		region:  MainRegionPublicPartition,
		profile: "old-profile",
	}

	err := provider.Init([]string{MainRegionPublicPartition})
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if !strings.Contains(err.Error(), "expected 2 init args") {
		t.Fatalf("Init error = %q, want missing AWS args", err)
	}
	if provider.region != "" {
		t.Fatalf("region = %q, want empty after failed init", provider.region)
	}
	if provider.profile != "" {
		t.Fatalf("profile = %q, want empty after failed init", provider.profile)
	}
}
