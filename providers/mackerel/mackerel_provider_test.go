// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"strings"
	"testing"
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
