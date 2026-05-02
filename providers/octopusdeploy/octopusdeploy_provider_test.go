// SPDX-License-Identifier: Apache-2.0

package octopusdeploy

import (
	"strings"
	"testing"
)

func TestOctopusDeployProviderInitReturnsMissingServerWithNoArgs(t *testing.T) {
	t.Setenv("OCTOPUS_CLI_SERVER", "")
	t.Setenv("OCTOPUS_CLI_API_KEY", "")
	var provider OctopusDeployProvider

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing server error")
	}
	if !strings.Contains(err.Error(), "server requirement") {
		t.Fatalf("expected server requirement error, got %q", err)
	}
}

func TestOctopusDeployProviderInitUsesEnvCredentials(t *testing.T) {
	t.Setenv("OCTOPUS_CLI_SERVER", "https://octopus.example.com")
	t.Setenv("OCTOPUS_CLI_API_KEY", "env-key")
	var provider OctopusDeployProvider

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.address != "https://octopus.example.com" {
		t.Fatalf("address = %q, want https://octopus.example.com", provider.address)
	}
	if provider.apiKey != "env-key" {
		t.Fatalf("apiKey = %q, want env-key", provider.apiKey)
	}
}

func TestOctopusDeployProviderInitPrefersArgs(t *testing.T) {
	t.Setenv("OCTOPUS_CLI_SERVER", "https://octopus.example.com")
	t.Setenv("OCTOPUS_CLI_API_KEY", "env-key")
	var provider OctopusDeployProvider

	if err := provider.Init([]string{"https://arg.example.com", "arg-key"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.address != "https://arg.example.com" {
		t.Fatalf("address = %q, want https://arg.example.com", provider.address)
	}
	if provider.apiKey != "arg-key" {
		t.Fatalf("apiKey = %q, want arg-key", provider.apiKey)
	}
}

func TestOctopusDeployProviderInitUsesEnvAPIKeyWithArgServer(t *testing.T) {
	t.Setenv("OCTOPUS_CLI_SERVER", "")
	t.Setenv("OCTOPUS_CLI_API_KEY", "env-key")
	var provider OctopusDeployProvider

	if err := provider.Init([]string{"https://arg.example.com"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.address != "https://arg.example.com" {
		t.Fatalf("address = %q, want https://arg.example.com", provider.address)
	}
	if provider.apiKey != "env-key" {
		t.Fatalf("apiKey = %q, want env-key", provider.apiKey)
	}
}
