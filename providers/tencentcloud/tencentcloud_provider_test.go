// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"strings"
	"testing"
)

func TestTencentCloudProviderInitRequiresRegion(t *testing.T) {
	var provider TencentCloudProvider

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing region error")
	}
	if !strings.Contains(err.Error(), "expected 1 init arg") {
		t.Fatalf("Init error = %q, want missing region", err)
	}
}
