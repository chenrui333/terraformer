// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"strings"
	"testing"
)

func TestAzureProviderInitRequiresResourceGroupArg(t *testing.T) {
	provider := AzureProvider{
		config: providerConfig{
			SubscriptionID: "old-subscription",
		},
		resourceGroup: "old-resource-group",
	}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing resource group arg error")
	}
	if !strings.Contains(err.Error(), "expected 1 init arg") {
		t.Fatalf("Init error = %q, want missing resource group arg", err)
	}
	if provider.resourceGroup != "" {
		t.Fatalf("resourceGroup = %q, want empty after failed init", provider.resourceGroup)
	}
	if provider.config.SubscriptionID != "" {
		t.Fatalf("SubscriptionID = %q, want empty after failed init", provider.config.SubscriptionID)
	}
}
