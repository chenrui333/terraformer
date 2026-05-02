// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type stubTokenCredential struct {
	token azcore.AccessToken
	err   error
	calls int
}

func (c *stubTokenCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
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

func TestSetEnvConfigPreservesCustomMetadataEnvironment(t *testing.T) {
	t.Setenv("ARM_SUBSCRIPTION_ID", "subscription-id")
	t.Setenv("ARM_ENVIRONMENT", "AzureStackCustomerCloud")
	t.Setenv("ARM_METADATA_HOSTNAME", "metadata.example.test")

	p := &AzureProvider{}
	if err := p.setEnvConfig(); err != nil {
		t.Fatalf("setEnvConfig() error = %v", err)
	}
	if p.config.Environment != "AzureStackCustomerCloud" {
		t.Fatalf("Environment = %q, want AzureStackCustomerCloud", p.config.Environment)
	}
}

func TestSetEnvConfigRejectsUnknownEnvironmentWithoutMetadataHost(t *testing.T) {
	t.Setenv("ARM_SUBSCRIPTION_ID", "subscription-id")
	t.Setenv("ARM_ENVIRONMENT", "AzureStackCustomerCloud")

	p := &AzureProvider{}
	if err := p.setEnvConfig(); err == nil {
		t.Fatal("setEnvConfig() error = nil, want unsupported environment error")
	}
}

func TestSetEnvConfigAcceptsGermanEnvironment(t *testing.T) {
	t.Setenv("ARM_SUBSCRIPTION_ID", "subscription-id")
	t.Setenv("ARM_ENVIRONMENT", "AzureGermanCloud")

	p := &AzureProvider{}
	if err := p.setEnvConfig(); err != nil {
		t.Fatalf("setEnvConfig() error = %v", err)
	}
	if p.config.Environment != "german" {
		t.Fatalf("Environment = %q, want german", p.config.Environment)
	}
}

func TestGetClientOptionsDisablesRPRegistration(t *testing.T) {
	p := &AzureProvider{}
	opts, err := p.getClientOptions()
	if err != nil {
		t.Fatalf("getClientOptions() error = %v", err)
	}
	if !opts.DisableRPRegistration {
		t.Fatal("DisableRPRegistration = false, want true")
	}
}

func TestGetClientOptionsGermanCloud(t *testing.T) {
	p := &AzureProvider{
		config: providerConfig{
			Environment: "german",
		},
	}
	opts, err := p.getClientOptions()
	if err != nil {
		t.Fatalf("getClientOptions() error = %v", err)
	}
	if opts.Cloud.ActiveDirectoryAuthorityHost != "https://login.microsoftonline.de/" {
		t.Fatalf("ActiveDirectoryAuthorityHost = %q, want German cloud login endpoint", opts.Cloud.ActiveDirectoryAuthorityHost)
	}
	resourceManager := opts.Cloud.Services[cloud.ResourceManager]
	if resourceManager.Endpoint != "https://management.microsoftazure.de" {
		t.Fatalf("ResourceManager endpoint = %q, want German cloud endpoint", resourceManager.Endpoint)
	}
	if resourceManager.Audience != "https://management.microsoftazure.de/" {
		t.Fatalf("ResourceManager audience = %q, want German cloud audience", resourceManager.Audience)
	}
}

func TestDiscoverCloudConfigFallsBackToMetadataHostResourceManager(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/endpoints" {
			t.Errorf("request path = %q, want /metadata/endpoints", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("api-version"); got != "2019-05-01" {
			t.Errorf("api-version = %q, want 2019-05-01", got)
			http.Error(w, "bad api-version", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[" +
			"{\"name\":\"AzureStackCustomerCloud\"," +
			"\"authentication\":{\"loginEndpoint\":\"https://login.example.test/\"," +
			"\"audiences\":[\"https://audience.example.test/\"]}," +
			"\"resourceManager\":\"\"}" +
			"]"))
	}))
	defer server.Close()

	oldClient := http.DefaultClient
	http.DefaultClient = server.Client()
	t.Cleanup(func() {
		http.DefaultClient = oldClient
	})

	cfg, err := discoverCloudConfig(strings.TrimPrefix(server.URL, "https://"), "AzureStackCustomerCloud")
	if err != nil {
		t.Fatalf("discoverCloudConfig() error = %v", err)
	}
	resourceManager := cfg.Services[cloud.ResourceManager]
	if resourceManager.Endpoint != server.URL+"/" {
		t.Fatalf("ResourceManager endpoint = %q, want metadata host fallback", resourceManager.Endpoint)
	}
	if resourceManager.Audience != "https://audience.example.test/" {
		t.Fatalf("ResourceManager audience = %q, want metadata audience", resourceManager.Audience)
	}
}

func TestDiscoverCloudConfigReturnsMetadataHTTPStatusErrors(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "metadata unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	oldClient := http.DefaultClient
	http.DefaultClient = server.Client()
	t.Cleanup(func() {
		http.DefaultClient = oldClient
	})

	_, err := discoverCloudConfig(strings.TrimPrefix(server.URL, "https://"), "")
	if err == nil {
		t.Fatal("expected metadata HTTP status error")
	}
	if !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Fatalf("expected status in error, got %q", err)
	}
	if !strings.Contains(err.Error(), "metadata unavailable") {
		t.Fatalf("expected response body in error, got %q", err)
	}
}

func TestAzureCLIUnavailableFallbackCredentialAllowsChainedFallback(t *testing.T) {
	for _, errMessage := range []string{
		"AzureCLICredential: Azure CLI not found on path",
		"AzureCLICredential: executable not found on path",
		"AzureCLICredential: Please run 'az login' to set up an account",
	} {
		t.Run(errMessage, func(t *testing.T) {
			fallbackToken := azcore.AccessToken{
				Token:     "fallback",
				ExpiresOn: time.Now().Add(time.Hour),
			}
			primary := &stubTokenCredential{err: errors.New(errMessage)}
			fallback := &stubTokenCredential{token: fallbackToken}
			chain, err := azidentity.NewChainedTokenCredential([]azcore.TokenCredential{
				azureCLIUnavailableFallbackCredential{credential: primary},
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
		})
	}
}

func TestAzureCLIUnavailableFallbackCredentialPreservesHardFailure(t *testing.T) {
	primary := &stubTokenCredential{err: errors.New("AzureCLICredential: AADSTS700082: the refresh token has expired; please run 'az login'")}
	fallback := &stubTokenCredential{token: azcore.AccessToken{
		Token:     "fallback",
		ExpiresOn: time.Now().Add(time.Hour),
	}}
	chain, err := azidentity.NewChainedTokenCredential([]azcore.TokenCredential{
		azureCLIUnavailableFallbackCredential{credential: primary},
		fallback,
	}, nil)
	if err != nil {
		t.Fatalf("creating credential chain: %v", err)
	}

	_, err = chain.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err == nil {
		t.Fatal("expected hard Azure CLI error, got nil")
	}
	if !strings.Contains(err.Error(), "refresh token has expired") {
		t.Fatalf("GetToken() error = %q, want refresh token failure", err)
	}
	if primary.calls != 1 {
		t.Fatalf("primary calls = %d, want 1", primary.calls)
	}
	if fallback.calls != 0 {
		t.Fatalf("fallback calls = %d, want 0", fallback.calls)
	}
}

func TestAzureCLIUnavailableFallbackCredentialKeepsSuccessfulPrimary(t *testing.T) {
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
		azureCLIUnavailableFallbackCredential{credential: primary},
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

func TestGetTokenCredentialPrefersClientCertificateOverSecret(t *testing.T) {
	p := &AzureProvider{
		clientOptions: &arm.ClientOptions{},
		config: providerConfig{
			ClientCertificatePath: t.TempDir() + "/missing.pem",
			ClientID:              "client-id",
			ClientSecret:          "stale-secret",
			TenantID:              "tenant-id",
			UseClientCertificate:  true,
			UseClientSecret:       true,
		},
	}

	_, err := p.getTokenCredential()
	if err == nil {
		t.Fatal("getTokenCredential() error = nil, want client certificate read error")
	}
	if !strings.Contains(err.Error(), "reading client certificate") {
		t.Fatalf("getTokenCredential() error = %q, want client certificate read error", err)
	}
}

func TestGetTokenCredentialSkipsPartialServicePrincipalConfig(t *testing.T) {
	t.Run("client secret", func(t *testing.T) {
		p := &AzureProvider{
			clientOptions: &arm.ClientOptions{},
			config: providerConfig{
				ClientSecret:    "leftover-secret",
				UseClientSecret: true,
			},
		}

		credential, err := p.getTokenCredential()
		if err != nil {
			t.Fatalf("getTokenCredential() error = %v", err)
		}
		if _, ok := credential.(*azidentity.ChainedTokenCredential); !ok {
			t.Fatalf("getTokenCredential() = %T, want fallback credential chain", credential)
		}
	})

	t.Run("client certificate", func(t *testing.T) {
		p := &AzureProvider{
			clientOptions: &arm.ClientOptions{},
			config: providerConfig{
				ClientCertificatePath: t.TempDir() + "/missing.pem",
				UseClientCertificate:  true,
			},
		}

		credential, err := p.getTokenCredential()
		if err != nil {
			t.Fatalf("getTokenCredential() error = %v", err)
		}
		if _, ok := credential.(*azidentity.ChainedTokenCredential); !ok {
			t.Fatalf("getTokenCredential() = %T, want fallback credential chain", credential)
		}
	})
}

func TestGetTokenCredentialPrefersOIDCOverManagedIdentity(t *testing.T) {
	p := &AzureProvider{
		clientOptions: &arm.ClientOptions{},
		config: providerConfig{
			ClientID:                      "client-id",
			CustomManagedIdentityEndpoint: "http://127.0.0.1/metadata/identity/oauth2/token",
			GitHubOIDCTokenRequestToken:   "request-token",
			GitHubOIDCTokenRequestURL:     "http://127.0.0.1/oidc",
			TenantID:                      "tenant-id",
			UseGitHubOIDC:                 true,
			UseManagedIdentity:            true,
		},
	}

	credential, err := p.getTokenCredential()
	if err != nil {
		t.Fatalf("getTokenCredential() error = %v", err)
	}
	if _, ok := credential.(*azidentity.ClientAssertionCredential); !ok {
		t.Fatalf("getTokenCredential() = %T, want *azidentity.ClientAssertionCredential", credential)
	}
}

func TestGetTokenCredentialSkipsOIDCWithoutRequestToken(t *testing.T) {
	p := &AzureProvider{
		clientOptions: &arm.ClientOptions{},
		config: providerConfig{
			ClientID:                      "client-id",
			CustomManagedIdentityEndpoint: "http://127.0.0.1/metadata/identity/oauth2/token",
			GitHubOIDCTokenRequestURL:     "http://127.0.0.1/oidc",
			TenantID:                      "tenant-id",
			UseGitHubOIDC:                 true,
			UseManagedIdentity:            true,
		},
	}

	credential, err := p.getTokenCredential()
	if err != nil {
		t.Fatalf("getTokenCredential() error = %v", err)
	}
	if _, ok := credential.(*customManagedIdentityCredential); !ok {
		t.Fatalf("getTokenCredential() = %T, want *customManagedIdentityCredential", credential)
	}
}

func TestGetTokenCredentialUsesCustomManagedIdentityEndpoint(t *testing.T) {
	t.Setenv("MSI_ENDPOINT", "")
	p := &AzureProvider{
		clientOptions: &arm.ClientOptions{},
		config: providerConfig{
			ClientID:                      "client-id",
			CustomManagedIdentityEndpoint: "http://127.0.0.1/metadata/identity/oauth2/token",
			UseManagedIdentity:            true,
		},
	}

	credential, err := p.getTokenCredential()
	if err != nil {
		t.Fatalf("getTokenCredential() error = %v", err)
	}
	if _, ok := credential.(*customManagedIdentityCredential); !ok {
		t.Fatalf("getTokenCredential() = %T, want *customManagedIdentityCredential", credential)
	}
	if got := os.Getenv("MSI_ENDPOINT"); got != "" {
		t.Fatalf("MSI_ENDPOINT = %q, want empty", got)
	}
}

func TestCustomManagedIdentityCredentialUsesMetadataGet(t *testing.T) {
	var (
		gotAPIVersion string
		gotClientID   string
		gotMetadata   string
		gotMethod     string
		gotResource   string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIVersion = r.URL.Query().Get("api-version")
		gotClientID = r.URL.Query().Get("client_id")
		gotMetadata = r.Header.Get("Metadata")
		gotMethod = r.Method
		gotResource = r.URL.Query().Get("resource")
		_, _ = w.Write([]byte("{\"access_token\":\"custom-msi-token\",\"expires_in\":\"3600\"}"))
	}))
	defer server.Close()

	credential := &customManagedIdentityCredential{
		clientID:   "client-id",
		endpoint:   server.URL + "/metadata/identity/oauth2/token",
		httpClient: server.Client(),
	}
	token, err := credential.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if token.Token != "custom-msi-token" {
		t.Fatalf("GetToken() token = %q, want custom-msi-token", token.Token)
	}
	if time.Until(token.ExpiresOn) <= 0 {
		t.Fatalf("GetToken() ExpiresOn = %s, want future timestamp", token.ExpiresOn)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotMetadata != "true" {
		t.Fatalf("Metadata header = %q, want true", gotMetadata)
	}
	if gotAPIVersion != "2018-02-01" {
		t.Fatalf("api-version = %q, want 2018-02-01", gotAPIVersion)
	}
	if gotResource != "https://management.azure.com" {
		t.Fatalf("resource = %q, want https://management.azure.com", gotResource)
	}
	if gotClientID != "client-id" {
		t.Fatalf("client_id = %q, want client-id", gotClientID)
	}
}

func TestGetGitHubOIDCAssertion(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var gotAudience string
		var gotAuthorization string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAudience = r.URL.Query().Get("audience")
			gotAuthorization = r.Header.Get("Authorization")
			_, _ = w.Write([]byte("{\"value\":\"assertion\"}"))
		}))
		defer server.Close()

		got, err := getGitHubOIDCAssertion(context.Background(), server.URL+"?existing=true", "request-token")
		if err != nil {
			t.Fatalf("getGitHubOIDCAssertion() error = %v", err)
		}
		if got != "assertion" {
			t.Fatalf("getGitHubOIDCAssertion() = %q, want assertion", got)
		}
		if gotAudience != "api://AzureADTokenExchange" {
			t.Fatalf("audience = %q, want api://AzureADTokenExchange", gotAudience)
		}
		if gotAuthorization != "Bearer request-token" {
			t.Fatalf("Authorization = %q, want Bearer request-token", gotAuthorization)
		}
	})

	t.Run("non-2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "{\"message\":\"bad token\"}", http.StatusUnauthorized)
		}))
		defer server.Close()

		_, err := getGitHubOIDCAssertion(context.Background(), server.URL, "request-token")
		if err == nil {
			t.Fatal("getGitHubOIDCAssertion() error = nil, want status error")
		}
		if !strings.Contains(err.Error(), "HTTP 401") {
			t.Fatalf("getGitHubOIDCAssertion() error = %q, want HTTP 401", err)
		}
	})

	t.Run("empty value", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("{}"))
		}))
		defer server.Close()

		_, err := getGitHubOIDCAssertion(context.Background(), server.URL, "request-token")
		if err == nil {
			t.Fatal("getGitHubOIDCAssertion() error = nil, want missing value error")
		}
		if !strings.Contains(err.Error(), "missing value") {
			t.Fatalf("getGitHubOIDCAssertion() error = %q, want missing value", err)
		}
	})
}
