// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"strings"
	"testing"
)

func TestAliCloudProviderInitRequiresArgs(t *testing.T) {
	var provider AliCloudProvider

	err := provider.Init([]string{"cn-hangzhou"})
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if !strings.Contains(err.Error(), "expected 2 init args") {
		t.Fatalf("Init error = %q, want missing AliCloud args", err)
	}
}
