// SPDX-License-Identifier: Apache-2.0

package opsgenie

import (
	"strings"
	"testing"
)

func TestOpsgenieProviderInitReturnsMissingAPIKeyWithNoArgs(t *testing.T) {
	t.Setenv("OPSGENIE_API_KEY", "")
	var provider OpsgenieProvider

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	if !strings.Contains(err.Error(), "required API Key missing") {
		t.Fatalf("expected missing API key error, got %q", err)
	}
}

func TestOpsgenieProviderInitDoesNotReuseStaleAPIKey(t *testing.T) {
	t.Setenv("OPSGENIE_API_KEY", "")
	provider := OpsgenieProvider{APIKey: "old-key"}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	if provider.APIKey != "" {
		t.Fatalf("APIKey = %q, want empty after failed init", provider.APIKey)
	}
}

func TestOpsgenieProviderInitUsesEnvAPIKey(t *testing.T) {
	t.Setenv("OPSGENIE_API_KEY", "env-key")
	var provider OpsgenieProvider

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.APIKey != "env-key" {
		t.Fatalf("APIKey = %q, want env-key", provider.APIKey)
	}
}

func TestOpsgenieProviderInitPrefersArgAPIKey(t *testing.T) {
	t.Setenv("OPSGENIE_API_KEY", "env-key")
	var provider OpsgenieProvider

	if err := provider.Init([]string{"arg-key"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.APIKey != "arg-key" {
		t.Fatalf("APIKey = %q, want arg-key", provider.APIKey)
	}
}
