// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"os"
	"strings"
	"testing"
)

func TestOpenStackProviderInitRequiresRegion(t *testing.T) {
	t.Setenv("OS_REGION_NAME", "old-region")
	provider := OpenStackProvider{region: "old-region"}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing region error")
	}
	if !strings.Contains(err.Error(), "expected 1 init arg") {
		t.Fatalf("Init error = %q, want missing region", err)
	}
	if provider.region != "" {
		t.Fatalf("region = %q, want empty after failed init", provider.region)
	}
	if value, ok := os.LookupEnv("OS_REGION_NAME"); ok {
		t.Fatalf("OS_REGION_NAME = %q, want unset after failed init", value)
	}
}

func TestOpenStackProviderInitReturnsRegionEnvError(t *testing.T) {
	var provider OpenStackProvider

	err := provider.Init([]string{"bad\x00region"})
	if err == nil {
		t.Fatal("expected region env error")
	}
	if !strings.Contains(err.Error(), `failed to set env OS_REGION_NAME="bad\x00region"`) {
		t.Fatalf("Init error = %q, want OS_REGION_NAME context", err)
	}
}
