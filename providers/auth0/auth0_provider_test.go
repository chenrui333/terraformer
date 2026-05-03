// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"strings"
	"testing"
)

func TestProviderInitRejectsInvalidDomain(t *testing.T) {
	t.Setenv("AUTH0_DOMAIN", "%zz")
	t.Setenv("AUTH0_CLIENT_ID", "client-id")
	t.Setenv("AUTH0_CLIENT_SECRET", "client-secret")

	provider := &Auth0Provider{}
	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected invalid domain to fail provider initialization")
	}
	if !strings.Contains(err.Error(), "create Auth0 management client") {
		t.Fatalf("expected management client error, got %q", err)
	}
}

func TestProviderInitServiceUsesInitializedClient(t *testing.T) {
	t.Setenv("AUTH0_DOMAIN", "example.auth0.com")
	t.Setenv("AUTH0_CLIENT_ID", "client-id")
	t.Setenv("AUTH0_CLIENT_SECRET", "client-secret")

	provider := &Auth0Provider{}
	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected provider initialization to succeed: %v", err)
	}
	if provider.client == nil {
		t.Fatal("expected provider initialization to store management client")
	}

	if err := provider.InitService("auth0_client", false); err != nil {
		t.Fatalf("expected service initialization to succeed: %v", err)
	}
	if provider.Service.GetArgs()[managementClientArg] != provider.client {
		t.Fatal("expected service to reuse provider-level management client")
	}
}

func TestProviderInitClearsStateOnMissingClientSecret(t *testing.T) {
	t.Setenv("AUTH0_DOMAIN", "example.auth0.com")
	t.Setenv("AUTH0_CLIENT_ID", "client-id")
	t.Setenv("AUTH0_CLIENT_SECRET", "client-secret")

	provider := &Auth0Provider{}
	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected provider initialization to succeed: %v", err)
	}
	if provider.domain != "example.auth0.com" || provider.clientID != "client-id" || provider.clientSecret != "client-secret" || provider.client == nil {
		t.Fatalf("expected provider state to be initialized, got domain=%q clientID=%q clientSecret=%q client=%v", provider.domain, provider.clientID, provider.clientSecret, provider.client)
	}

	t.Setenv("AUTH0_CLIENT_SECRET", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected provider initialization to fail without AUTH0_CLIENT_SECRET")
	}
	if provider.domain != "" || provider.clientID != "" || provider.clientSecret != "" || provider.client != nil {
		t.Fatalf("expected stale provider state to be cleared, got domain=%q clientID=%q clientSecret=%q client=%v", provider.domain, provider.clientID, provider.clientSecret, provider.client)
	}
}
