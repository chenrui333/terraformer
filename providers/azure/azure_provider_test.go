// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"strings"
	"testing"
)

func TestAzureProviderInitRequiresResourceGroupArg(t *testing.T) {
	var provider AzureProvider

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing resource group arg error")
	}
	if !strings.Contains(err.Error(), "expected 1 init arg") {
		t.Fatalf("Init error = %q, want missing resource group arg", err)
	}
}
