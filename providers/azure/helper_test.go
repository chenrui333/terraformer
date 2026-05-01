// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type stubTokenCredential struct {
	token azcore.AccessToken
	err   error
	calls int
}

func (c *stubTokenCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.calls++
	return c.token, c.err
}

func TestParseAzureResourceID(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		wantSubID     string
		wantRG        string
		wantProvider  string
		wantPathKey   string
		wantPathValue string
		wantErr       bool
	}{
		{
			name:          "standard resource",
			id:            "/subscriptions/sub-123/resourceGroups/my-rg/providers/Microsoft.Network/virtualNetworks/my-vnet",
			wantSubID:     "sub-123",
			wantRG:        "my-rg",
			wantProvider:  "Microsoft.Network",
			wantPathKey:   "virtualNetworks",
			wantPathValue: "my-vnet",
		},
		{
			name:          "storage account",
			id:            "/subscriptions/sub-456/resourceGroups/prod-rg/providers/Microsoft.Storage/storageAccounts/mystorage",
			wantSubID:     "sub-456",
			wantRG:        "prod-rg",
			wantProvider:  "Microsoft.Storage",
			wantPathKey:   "storageAccounts",
			wantPathValue: "mystorage",
		},
		{
			name:         "lowercase resourcegroups",
			id:           "/subscriptions/sub-789/resourcegroups/lower-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
			wantSubID:    "sub-789",
			wantRG:       "lower-rg",
			wantProvider: "Microsoft.Compute",
		},
		{
			name:      "resource group only",
			id:        "/subscriptions/sub-123/resourceGroups/my-rg",
			wantSubID: "sub-123",
			wantRG:    "my-rg",
		},
		{
			name:    "invalid url",
			id:      "not-a-url",
			wantErr: true,
		},
		{
			name:    "odd number of segments",
			id:      "/subscriptions/sub-123/resourceGroups",
			wantErr: true,
		},
		{
			name:    "no subscription",
			id:      "/resourceGroups/my-rg/providers/Microsoft.Network/virtualNetworks/my-vnet",
			wantErr: true,
		},
		{
			name:    "empty segment",
			id:      "/subscriptions//resourceGroups/my-rg",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAzureResourceID(tc.id)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.SubscriptionID != tc.wantSubID {
				t.Errorf("SubscriptionID = %q, want %q", got.SubscriptionID, tc.wantSubID)
			}
			if got.ResourceGroup != tc.wantRG {
				t.Errorf("ResourceGroup = %q, want %q", got.ResourceGroup, tc.wantRG)
			}
			if tc.wantProvider != "" && got.Provider != tc.wantProvider {
				t.Errorf("Provider = %q, want %q", got.Provider, tc.wantProvider)
			}
			if tc.wantPathKey != "" {
				if v, ok := got.Path[tc.wantPathKey]; !ok || v != tc.wantPathValue {
					t.Errorf("Path[%q] = %q, want %q", tc.wantPathKey, v, tc.wantPathValue)
				}
			}
		})
	}
}

func TestAsHereDoc(t *testing.T) {
	got := asHereDoc(`{"key":"value"}`)
	want := "<<JSON\n{\"key\":\"value\"}\nJSON"
	if got != want {
		t.Errorf("asHereDoc() = %q, want %q", got, want)
	}
}

func TestCredentialUnavailableOnErrorAllowsChainedFallback(t *testing.T) {
	fallbackToken := azcore.AccessToken{
		Token:     "fallback",
		ExpiresOn: time.Now().Add(time.Hour),
	}
	primary := &stubTokenCredential{err: errors.New("Azure CLI unavailable")}
	fallback := &stubTokenCredential{token: fallbackToken}
	chain, err := azidentity.NewChainedTokenCredential([]azcore.TokenCredential{
		credentialUnavailableOnError{credential: primary},
		fallback,
	}, nil)
	if err != nil {
		t.Fatalf("creating credential chain: %v", err)
	}

	got, err := chain.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		t.Fatalf("expected fallback token, got error: %v", err)
	}
	if got.Token != fallbackToken.Token {
		t.Fatalf("GetToken() token = %q, want %q", got.Token, fallbackToken.Token)
	}
	if primary.calls != 1 {
		t.Fatalf("primary calls = %d, want 1", primary.calls)
	}
	if fallback.calls != 1 {
		t.Fatalf("fallback calls = %d, want 1", fallback.calls)
	}
}

func TestCredentialUnavailableOnErrorKeepsSuccessfulPrimary(t *testing.T) {
	primaryToken := azcore.AccessToken{
		Token:     "primary",
		ExpiresOn: time.Now().Add(time.Hour),
	}
	primary := &stubTokenCredential{token: primaryToken}
	fallback := &stubTokenCredential{token: azcore.AccessToken{
		Token:     "fallback",
		ExpiresOn: time.Now().Add(time.Hour),
	}}
	chain, err := azidentity.NewChainedTokenCredential([]azcore.TokenCredential{
		credentialUnavailableOnError{credential: primary},
		fallback,
	}, nil)
	if err != nil {
		t.Fatalf("creating credential chain: %v", err)
	}

	got, err := chain.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		t.Fatalf("expected primary token, got error: %v", err)
	}
	if got.Token != primaryToken.Token {
		t.Fatalf("GetToken() token = %q, want %q", got.Token, primaryToken.Token)
	}
	if primary.calls != 1 {
		t.Fatalf("primary calls = %d, want 1", primary.calls)
	}
	if fallback.calls != 0 {
		t.Fatalf("fallback calls = %d, want 0", fallback.calls)
	}
}
