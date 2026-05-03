// SPDX-License-Identifier: Apache-2.0

package panos

import (
	"strings"
	"testing"
)

func TestPanosProviderInitRequiresArgs(t *testing.T) {
	provider := PanosProvider{vsys: "old-vsys"}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if !strings.Contains(err.Error(), "vsys is required") {
		t.Fatalf("Init error = %q, want missing PAN-OS args", err)
	}
	if provider.vsys != "" {
		t.Fatalf("vsys = %q, want empty after failed init", provider.vsys)
	}
}

func TestPanosProviderInitStoresArgs(t *testing.T) {
	var provider PanosProvider

	if err := provider.Init([]string{"vsys1"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.vsys != "vsys1" {
		t.Fatalf("vsys = %q, want vsys1", provider.vsys)
	}
}
