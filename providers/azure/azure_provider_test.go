// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
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

func TestAzureProviderInitClearsStateOnLaterInitError(t *testing.T) {
	t.Setenv("ARM_SUBSCRIPTION_ID", "subscription-id")
	t.Setenv("ARM_CLIENT_ID", "client-id")
	t.Setenv("ARM_TENANT_ID", "tenant-id")
	t.Setenv("ARM_CLIENT_CERTIFICATE_PATH", t.TempDir()+"/missing.pem")

	provider := AzureProvider{
		config: providerConfig{
			SubscriptionID:       "old-subscription",
			UseClientCertificate: true,
		},
		credential:    &stubTokenCredential{},
		clientOptions: &arm.ClientOptions{},
		resourceGroup: "old-resource-group",
	}

	err := provider.Init([]string{"resource-group"})
	if err == nil {
		t.Fatal("expected client certificate error")
	}
	if !strings.Contains(err.Error(), "reading client certificate") {
		t.Fatalf("Init error = %q, want client certificate read error", err)
	}
	if provider.resourceGroup != "" {
		t.Fatalf("resourceGroup = %q, want empty after failed init", provider.resourceGroup)
	}
	if provider.config.SubscriptionID != "" {
		t.Fatalf("SubscriptionID = %q, want empty after failed init", provider.config.SubscriptionID)
	}
	if provider.config.UseClientCertificate {
		t.Fatal("UseClientCertificate = true, want false after failed init")
	}
	if provider.credential != nil {
		t.Fatalf("credential = %T, want nil after failed init", provider.credential)
	}
	if provider.clientOptions != nil {
		t.Fatal("clientOptions should be nil after failed init")
	}
}
