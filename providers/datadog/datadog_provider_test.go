// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"strings"
	"testing"
)

func TestDatadogProviderInitHandlesShortArgs(t *testing.T) {
	t.Setenv("DATADOG_API_KEY", "")
	t.Setenv("DATADOG_APP_KEY", "")
	t.Setenv("DATADOG_HOST", "")
	t.Setenv("DATADOG_VALIDATE", "false")

	var provider DatadogProvider
	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected Init to accept missing optional args with validation disabled: %v", err)
	}
	if provider.validate {
		t.Fatal("validate = true, want false")
	}
}

func TestDatadogProviderInitClearsStaleOptionalState(t *testing.T) {
	t.Setenv("DATADOG_API_KEY", "")
	t.Setenv("DATADOG_APP_KEY", "")
	t.Setenv("DATADOG_HOST", "")
	t.Setenv("DATADOG_VALIDATE", "false")
	provider := DatadogProvider{
		apiKey:   "old-api-key",
		appKey:   "old-app-key",
		apiURL:   "https://old.example.com",
		validate: true,
	}

	if err := provider.Init([]string{"", "", "", ""}); err != nil {
		t.Fatalf("expected Init to accept empty args with validation disabled: %v", err)
	}
	if provider.apiKey != "" {
		t.Fatalf("apiKey = %q, want empty", provider.apiKey)
	}
	if provider.appKey != "" {
		t.Fatalf("appKey = %q, want empty", provider.appKey)
	}
	if provider.apiURL != "" {
		t.Fatalf("apiURL = %q, want empty", provider.apiURL)
	}
	if provider.validate {
		t.Fatal("validate = true, want false")
	}
}

func TestDatadogProviderInitReturnsCredentialErrorForShortArgs(t *testing.T) {
	t.Setenv("DATADOG_API_KEY", "")
	t.Setenv("DATADOG_APP_KEY", "")
	t.Setenv("DATADOG_HOST", "")
	t.Setenv("DATADOG_VALIDATE", "")

	var provider DatadogProvider
	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	if !strings.Contains(err.Error(), "api-key requirement") {
		t.Fatalf("Init error = %q, want missing API key", err)
	}
}

func TestDatadogProviderInitClearsStateOnValidateError(t *testing.T) {
	t.Setenv("DATADOG_API_KEY", "env-api-key")
	t.Setenv("DATADOG_APP_KEY", "env-app-key")
	t.Setenv("DATADOG_HOST", "https://old.example.com")
	t.Setenv("DATADOG_VALIDATE", "not-bool")
	provider := DatadogProvider{
		apiKey:   "old-api-key",
		appKey:   "old-app-key",
		apiURL:   "https://stale.example.com",
		validate: true,
	}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected invalid validate error")
	}
	if !strings.Contains(err.Error(), "invalid DATADOG_VALIDATE") {
		t.Fatalf("Init error = %q, want validate parse error", err)
	}
	if provider.apiKey != "" {
		t.Fatalf("apiKey = %q, want empty after failed init", provider.apiKey)
	}
	if provider.appKey != "" {
		t.Fatalf("appKey = %q, want empty after failed init", provider.appKey)
	}
	if provider.apiURL != "" {
		t.Fatalf("apiURL = %q, want empty after failed init", provider.apiURL)
	}
	if provider.validate {
		t.Fatal("validate = true, want false after failed init")
	}
	if provider.auth != nil {
		t.Fatal("auth is set, want nil after failed init")
	}
	if provider.datadogClient != nil {
		t.Fatal("datadogClient is set, want nil after failed init")
	}
}

func TestDatadogProviderAPMRetentionFilterConnections(t *testing.T) {
	connections := DatadogProvider{}.GetResourceConnections()

	assertDatadogConnection(
		t,
		connections,
		"apm_retention_filter_order",
		"apm_retention_filter",
		"filter_ids",
		"id",
	)
}

func TestDatadogProviderRumRetentionFilterConnections(t *testing.T) {
	connections := DatadogProvider{}.GetResourceConnections()

	assertDatadogConnection(
		t,
		connections,
		"rum_retention_filter",
		"rum_application",
		"application_id",
		"id",
	)
	assertDatadogConnection(
		t,
		connections,
		"rum_retention_filters_order",
		"rum_application",
		"application_id",
		"id",
	)
}

func TestDatadogProviderSensitiveDataScannerConnections(t *testing.T) {
	connections := DatadogProvider{}.GetResourceConnections()

	assertDatadogConnection(
		t,
		connections,
		"sensitive_data_scanner_group_order",
		"sensitive_data_scanner_group",
		"group_ids",
		"id",
	)
	assertDatadogConnection(
		t,
		connections,
		"sensitive_data_scanner_rule",
		"sensitive_data_scanner_group",
		"group_id",
		"id",
	)
}

func TestDatadogProviderTeamRelationshipConnections(t *testing.T) {
	connections := DatadogProvider{}.GetResourceConnections()

	assertDatadogConnection(
		t,
		connections,
		"team_connection",
		"team",
		"team.id",
		"id",
	)
	assertDatadogConnectionPairs(
		t,
		connections,
		"team_hierarchy_links",
		"team",
		[]string{
			"parent_team_id", "id",
			"sub_team_id", "id",
		},
	)
}

func TestDatadogProviderMonitorJSONConnections(t *testing.T) {
	connections := DatadogProvider{}.GetResourceConnections()

	assertDatadogConnectionPairs(
		t,
		connections,
		"dashboard",
		"monitor_json",
		[]string{
			"widget.alert_graph_definition.alert_id", "id",
			"widget.group_definition.widget.alert_graph_definition.alert_id", "id",
			"widget.alert_value_definition.alert_id", "id",
			"widget.group_definition.widget.alert_value_definition.alert_id", "id",
		},
	)
	assertDatadogConnection(
		t,
		connections,
		"downtime",
		"monitor_json",
		"monitor_id",
		"id",
	)
	assertDatadogConnection(
		t,
		connections,
		"downtime_schedule",
		"monitor_json",
		"monitor_identifier.monitor_id",
		"id",
	)
	assertDatadogConnection(
		t,
		connections,
		"service_level_objective",
		"monitor_json",
		"monitor_ids",
		"id",
	)
}

func TestDatadogProviderDowntimeScheduleConnections(t *testing.T) {
	connections := DatadogProvider{}.GetResourceConnections()

	assertDatadogConnection(
		t,
		connections,
		"downtime_schedule",
		"monitor",
		"monitor_identifier.monitor_id",
		"id",
	)
}

func TestDatadogProviderSyntheticsSuiteConnections(t *testing.T) {
	connections := DatadogProvider{}.GetResourceConnections()

	assertDatadogConnection(
		t,
		connections,
		"synthetics_suite",
		"synthetics_test",
		"tests.public_id",
		"id",
	)
}

func assertDatadogConnection(t *testing.T, connections map[string]map[string][]string, source, target, sourceField, targetField string) {
	t.Helper()

	assertDatadogConnectionPairs(t, connections, source, target, []string{sourceField, targetField})
}

func assertDatadogConnectionPairs(t *testing.T, connections map[string]map[string][]string, source, target string, expected []string) {
	t.Helper()

	targets, ok := connections[source]
	if !ok {
		t.Fatalf("connections[%q] missing", source)
	}
	fields, ok := targets[target]
	if !ok {
		t.Fatalf("connections[%q][%q] missing", source, target)
	}
	if len(fields) != len(expected) {
		t.Fatalf("connections[%q][%q] = %v, want %v", source, target, fields, expected)
	}
	for i := range expected {
		if fields[i] != expected[i] {
			t.Fatalf("connections[%q][%q] = %v, want %v", source, target, fields, expected)
		}
	}
}
