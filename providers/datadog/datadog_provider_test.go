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
		"service_level_objective",
		"monitor_json",
		"monitor_ids",
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
