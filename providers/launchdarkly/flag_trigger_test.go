// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v16"
)

func TestGetTriggerWorkflowsUsesSDKArgumentOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/api/v2/flags/proj/flag-key/triggers/prod"
		if r.URL.Path != wantPath {
			t.Fatalf("request path = %q, want %q", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"items\":[{\"_id\":\"trigger-1\",\"_integrationKey\":\"generic\"}]}"))
	}))
	defer server.Close()

	config := ldapi.NewConfiguration()
	config.Servers = ldapi.ServerConfigurations{{URL: server.URL}}
	client := ldapi.NewAPIClient(config)

	triggers, resp, err := getTriggerWorkflows(context.Background(), client, "proj", "prod", "flag-key")
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("getTriggerWorkflows() error = %v", err)
	}
	items := triggers.GetItems()
	if len(items) != 1 || items[0].GetId() != "trigger-1" {
		t.Fatalf("getTriggerWorkflows() items = %#v", items)
	}
}
