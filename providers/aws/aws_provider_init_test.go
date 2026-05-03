// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"os"
	"strings"
	"testing"
)

func TestAWSProviderInitRequiresArgs(t *testing.T) {
	t.Setenv("AWS_REGION", "old-region")
	t.Setenv("AWS_DEFAULT_REGION", "old-default-region")
	t.Setenv("AWS_PROFILE", "old-profile")
	t.Setenv("AWS_DEFAULT_PROFILE", "old-default-profile")
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
	for _, key := range []string{"AWS_REGION", "AWS_DEFAULT_REGION", "AWS_PROFILE", "AWS_DEFAULT_PROFILE"} {
		if value, ok := os.LookupEnv(key); ok {
			t.Fatalf("%s = %q, want unset after failed init", key, value)
		}
	}
}

func TestAWSProviderInitPreservesAmbientRegionForNoRegion(t *testing.T) {
	t.Setenv("AWS_REGION", "env-region")
	t.Setenv("AWS_DEFAULT_REGION", "env-default-region")
	t.Setenv("AWS_PROFILE", "old-profile")
	t.Setenv("AWS_DEFAULT_PROFILE", "old-default-profile")
	var provider AWSProvider

	if err := provider.Init([]string{NoRegion, "default"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.region != NoRegion {
		t.Fatalf("region = %q, want NoRegion", provider.region)
	}
	if got := os.Getenv("AWS_REGION"); got != "env-region" {
		t.Fatalf("AWS_REGION = %q, want env-region", got)
	}
	if got := os.Getenv("AWS_DEFAULT_REGION"); got != "env-default-region" {
		t.Fatalf("AWS_DEFAULT_REGION = %q, want env-default-region", got)
	}
	if value, ok := os.LookupEnv("AWS_PROFILE"); ok {
		t.Fatalf("AWS_PROFILE = %q, want unset", value)
	}
	if value, ok := os.LookupEnv("AWS_DEFAULT_PROFILE"); ok {
		t.Fatalf("AWS_DEFAULT_PROFILE = %q, want unset", value)
	}
}
