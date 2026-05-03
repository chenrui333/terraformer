// SPDX-License-Identifier: Apache-2.0

package honeycombio

import (
	"strings"
	"testing"
)

func TestHoneycombProviderInitClearsStateOnMissingAPIKey(t *testing.T) {
	t.Setenv("HONEYCOMB_API_KEY", "")
	t.Setenv("HONEYCOMB_API_URL", "https://api.eu1.honeycomb.io")
	provider := HoneycombProvider{
		apiKey:   "old-key",
		apiURL:   "https://old.example.com",
		datasets: []string{"old-dataset"},
	}

	err := provider.Init([]string{"dataset"})
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	if !strings.Contains(err.Error(), "HONEYCOMB_API_KEY") {
		t.Fatalf("Init error = %q, want API key requirement", err)
	}
	if provider.apiKey != "" {
		t.Fatalf("apiKey = %q, want empty after failed init", provider.apiKey)
	}
	if provider.apiURL != "" {
		t.Fatalf("apiURL = %q, want empty after failed init", provider.apiURL)
	}
	if provider.datasets != nil {
		t.Fatalf("datasets = %v, want nil after failed init", provider.datasets)
	}
}

func TestHoneycombProviderInitUsesDefaultAPIURL(t *testing.T) {
	t.Setenv("HONEYCOMB_API_KEY", "api-key")
	t.Setenv("HONEYCOMB_API_URL", "")
	var provider HoneycombProvider

	if err := provider.Init([]string{"dataset"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.apiKey != "api-key" {
		t.Fatalf("apiKey = %q, want api-key", provider.apiKey)
	}
	if provider.apiURL != honeycombDefaultURL {
		t.Fatalf("apiURL = %q, want %q", provider.apiURL, honeycombDefaultURL)
	}
	if len(provider.datasets) != 1 || provider.datasets[0] != "dataset" {
		t.Fatalf("datasets = %v, want [dataset]", provider.datasets)
	}
}

func TestHoneycombDebugEnabledRejectsInvalidValue(t *testing.T) {
	t.Setenv("HONEYCOMBIO_DEBUG", "sometimes")

	_, err := honeycombDebugEnabled()
	if err == nil {
		t.Fatal("expected invalid HONEYCOMBIO_DEBUG error")
	}
	if !strings.Contains(err.Error(), "HONEYCOMBIO_DEBUG") {
		t.Fatalf("error = %q, want HONEYCOMBIO_DEBUG context", err)
	}
}

func TestHoneycombDebugEnabledParsesValue(t *testing.T) {
	t.Setenv("HONEYCOMBIO_DEBUG", "true")

	enabled, err := honeycombDebugEnabled()
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if !enabled {
		t.Fatal("enabled = false, want true")
	}
}
