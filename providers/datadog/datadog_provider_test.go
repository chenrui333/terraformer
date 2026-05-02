// SPDX-License-Identifier: Apache-2.0

package datadog

import "testing"

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

func assertDatadogConnection(t *testing.T, connections map[string]map[string][]string, source, target, sourceField, targetField string) {
	t.Helper()

	targets, ok := connections[source]
	if !ok {
		t.Fatalf("connections[%q] missing", source)
	}
	fields, ok := targets[target]
	if !ok {
		t.Fatalf("connections[%q][%q] missing", source, target)
	}
	if len(fields) != 2 {
		t.Fatalf("connections[%q][%q] = %v, want two fields", source, target, fields)
	}
	if fields[0] != sourceField || fields[1] != targetField {
		t.Fatalf("connections[%q][%q] = %v, want [%q %q]", source, target, fields, sourceField, targetField)
	}
}
