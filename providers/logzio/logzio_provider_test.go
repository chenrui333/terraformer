// SPDX-License-Identifier: Apache-2.0

package logzio

import (
	"strings"
	"testing"
)

func TestLogzioProviderInitRequiresArgs(t *testing.T) {
	provider := LogzioProvider{
		apiToken: "old-token",
		baseURL:  "https://old.example.com",
	}

	if err := provider.Init([]string{"token"}); err == nil {
		t.Fatal("expected missing base URL error")
	}
	if provider.apiToken != "" {
		t.Fatalf("apiToken = %q, want empty after failed init", provider.apiToken)
	}
	if provider.baseURL != "" {
		t.Fatalf("baseURL = %q, want empty after failed init", provider.baseURL)
	}
}

func TestLogzioProviderInitStoresArgs(t *testing.T) {
	var provider LogzioProvider

	if err := provider.Init([]string{"token", "https://api.logz.io"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.apiToken != "token" {
		t.Fatalf("apiToken = %q, want token", provider.apiToken)
	}
	if provider.baseURL != "https://api.logz.io" {
		t.Fatalf("baseURL = %q, want https://api.logz.io", provider.baseURL)
	}
}

func TestAlertsGeneratorReturnsClientError(t *testing.T) {
	generator := &AlertsGenerator{}
	generator.SetArgs(map[string]interface{}{
		"api_token": "",
		"base_url":  "https://api.logz.io",
	})

	err := generator.InitResources()
	if err == nil {
		t.Fatal("expected client setup error")
	}
	if !strings.Contains(err.Error(), "API token not defined") {
		t.Fatalf("expected API token error, got %q", err)
	}
}

func TestAlertNotificationEndpointsGeneratorReturnsClientError(t *testing.T) {
	generator := &AlertNotificationEndpointsGenerator{}
	generator.SetArgs(map[string]interface{}{
		"api_token": "token",
		"base_url":  "",
	})

	err := generator.InitResources()
	if err == nil {
		t.Fatal("expected client setup error")
	}
	if !strings.Contains(err.Error(), "Base URL not defined") {
		t.Fatalf("expected base URL error, got %q", err)
	}
}
