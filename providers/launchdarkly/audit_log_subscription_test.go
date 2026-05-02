// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"reflect"
	"testing"
)

func TestAuditLogSubscriptionAttributesSeedsConfig(t *testing.T) {
	got := auditLogSubscriptionAttributes("datadog", map[string]interface{}{
		"apiKey":          "secret",
		"hostURL":         "https://api.datadoghq.com",
		"skipHTTPArchive": true,
		"nested":          map[string]interface{}{"key": "value"},
	})

	want := map[string]string{
		"integration_key":          "datadog",
		"config.%":                 "4",
		"config.api_key":           "secret",
		"config.host_url":          "https://api.datadoghq.com",
		"config.skip_http_archive": "true",
		"config.nested":            `{"key":"value"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("auditLogSubscriptionAttributes() = %#v, want %#v", got, want)
	}
}
