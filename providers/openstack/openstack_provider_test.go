// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"strings"
	"testing"
)

func TestOpenStackProviderInitRequiresRegion(t *testing.T) {
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
}
