// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"reflect"
	"testing"
)

func TestAuditLogSubscriptionResourceNameIncludesSubscriptionID(t *testing.T) {
	got := auditLogSubscriptionResourceName("datadog", "duplicate", "sub-123")
	want := "datadog-duplicate-sub-123"
	if got != want {
		t.Fatalf("auditLogSubscriptionResourceName() = %q, want %q", got, want)
	}
}

func TestAuditLogSubscriptionAttributesSeedsConfig(t *testing.T) {
	got := auditLogSubscriptionAttributes("datadog", map[string]interface{}{
		"apiKey":          "secret",
		"hostURL":         "https://api.datadoghq.com",
		"last9":           "token",
		"skipHTTPArchive": true,
		"nested":          map[string]interface{}{"key": "value"},
	})

	want := map[string]string{
		"integration_key":          "datadog",
		"config.%":                 "5",
		"config.api_key":           "secret",
		"config.host_url":          "https://api.datadoghq.com",
		"config.last9":             "token",
		"config.skip_http_archive": "true",
		"config.nested":            `{"key":"value"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("auditLogSubscriptionAttributes() = %#v, want %#v", got, want)
	}
}
