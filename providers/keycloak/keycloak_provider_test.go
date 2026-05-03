// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"strings"
	"testing"
)

func validKeycloakInitArgs() []string {
	return []string{
		"https://keycloak.example.com",
		"/auth",
		"terraformer",
		"secret",
		"master",
		"30",
		"-",
		"true",
		"false",
		"-",
	}
}

func TestKeycloakProviderInitRequiresArgs(t *testing.T) {
	provider := KeycloakProvider{
		url:                   "https://old.example.com",
		basePath:              "/old",
		clientID:              "old-client",
		clientSecret:          "old-secret",
		realm:                 "old-realm",
		clientTimeout:         99,
		caCert:                "old-cert",
		tlsInsecureSkipVerify: true,
		redHatSSO:             true,
		target:                "old-target",
	}

	err := provider.Init(validKeycloakInitArgs()[:9])
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if !strings.Contains(err.Error(), "expected 10 init args") {
		t.Fatalf("expected init arg count error, got %q", err)
	}
	assertKeycloakProviderCleared(t, provider)
}

func TestKeycloakProviderInitReturnsClientTimeoutError(t *testing.T) {
	provider := KeycloakProvider{url: "https://old.example.com", clientTimeout: 30}
	args := validKeycloakInitArgs()
	args[5] = "slow"

	err := provider.Init(args)
	if err == nil {
		t.Fatal("expected client timeout parse error")
	}
	if !strings.Contains(err.Error(), "invalid client timeout") {
		t.Fatalf("expected client timeout error, got %q", err)
	}
	assertKeycloakProviderCleared(t, provider)
}

func TestKeycloakProviderInitReturnsTLSBoolError(t *testing.T) {
	provider := KeycloakProvider{url: "https://old.example.com", tlsInsecureSkipVerify: true}
	args := validKeycloakInitArgs()
	args[7] = "sometimes"

	err := provider.Init(args)
	if err == nil {
		t.Fatal("expected TLS bool parse error")
	}
	if !strings.Contains(err.Error(), "invalid tls insecure skip verify") {
		t.Fatalf("expected TLS bool error, got %q", err)
	}
	assertKeycloakProviderCleared(t, provider)
}

func TestKeycloakProviderInitReturnsRedHatSSOBoolError(t *testing.T) {
	provider := KeycloakProvider{url: "https://old.example.com", redHatSSO: true}
	args := validKeycloakInitArgs()
	args[8] = "maybe"

	err := provider.Init(args)
	if err == nil {
		t.Fatal("expected Red Hat SSO bool parse error")
	}
	if !strings.Contains(err.Error(), "invalid red hat sso") {
		t.Fatalf("expected Red Hat SSO bool error, got %q", err)
	}
	assertKeycloakProviderCleared(t, provider)
}

func TestKeycloakProviderInitStoresArgs(t *testing.T) {
	var provider KeycloakProvider

	if err := provider.Init(validKeycloakInitArgs()); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.url != "https://keycloak.example.com" {
		t.Fatalf("url = %q, want https://keycloak.example.com", provider.url)
	}
	if provider.clientTimeout != 30 {
		t.Fatalf("clientTimeout = %d, want 30", provider.clientTimeout)
	}
	if provider.caCert != "" {
		t.Fatalf("caCert = %q, want empty", provider.caCert)
	}
	if !provider.tlsInsecureSkipVerify {
		t.Fatal("tlsInsecureSkipVerify = false, want true")
	}
	if provider.redHatSSO {
		t.Fatal("redHatSSO = true, want false")
	}
	if provider.target != "" {
		t.Fatalf("target = %q, want empty", provider.target)
	}
}

func assertKeycloakProviderCleared(t *testing.T, provider KeycloakProvider) {
	t.Helper()

	if provider.url != "" || provider.basePath != "" || provider.clientID != "" ||
		provider.clientSecret != "" || provider.realm != "" || provider.clientTimeout != 0 ||
		provider.caCert != "" || provider.tlsInsecureSkipVerify || provider.redHatSSO ||
		provider.target != "" {
		t.Fatalf("provider state was not cleared after failed init: %#v", provider)
	}
}
