// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"strings"
	"testing"

	mackerelapi "github.com/mackerelio/mackerel-client-go"
)

func TestMackerelProviderInitReturnsMissingAPIKeyWithNoArgs(t *testing.T) {
	t.Setenv("MACKEREL_API_KEY", "")
	var provider MackerelProvider

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	if !strings.Contains(err.Error(), "api-key requirement") {
		t.Fatalf("expected missing API key error, got %q", err)
	}
}

func TestMackerelProviderInitClearsStateOnMissingAPIKey(t *testing.T) {
	t.Setenv("MACKEREL_API_KEY", "")
	provider := MackerelProvider{
		apiKey:         "old-key",
		mackerelClient: mackerelapi.NewClient("old-key"),
	}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	if provider.apiKey != "" {
		t.Fatalf("apiKey = %q, want empty", provider.apiKey)
	}
	if provider.mackerelClient != nil {
		t.Fatal("mackerelClient should be nil after failed init")
	}
}

func TestMackerelProviderInitUsesEnvAPIKey(t *testing.T) {
	t.Setenv("MACKEREL_API_KEY", "env-key")
	var provider MackerelProvider

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.apiKey != "env-key" {
		t.Fatalf("apiKey = %q, want env-key", provider.apiKey)
	}
	if provider.mackerelClient == nil {
		t.Fatal("expected mackerel client to be initialized")
	}
}

func TestMackerelProviderInitPrefersArgAPIKey(t *testing.T) {
	t.Setenv("MACKEREL_API_KEY", "env-key")
	var provider MackerelProvider

	if err := provider.Init([]string{"arg-key"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.apiKey != "arg-key" {
		t.Fatalf("apiKey = %q, want arg-key", provider.apiKey)
	}
}
