// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestServiceAccountApplicationKeyCreateResource(t *testing.T) {
	generator := &ServiceAccountApplicationKeyGenerator{}
	resource, err := generator.createResource("sa-123", "key-456")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "key-456" {
		t.Fatalf("expected resource ID key-456, got %s", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != "datadog_service_account_application_key" {
		t.Fatalf("expected resource type datadog_service_account_application_key, got %s", resource.InstanceInfo.Type)
	}
}

func TestServiceAccountApplicationKeyCreateResourceEmptyKeyID(t *testing.T) {
	generator := &ServiceAccountApplicationKeyGenerator{}
	_, err := generator.createResource("sa-123", "")
	if err == nil {
		t.Fatal("expected error for empty key ID, got nil")
	}
}

func TestServiceAccountApplicationKeyCreateResourceEmptyServiceAccountID(t *testing.T) {
	generator := &ServiceAccountApplicationKeyGenerator{}
	_, err := generator.createResource("", "key-456")
	if err == nil {
		t.Fatal("expected error for empty service account ID, got nil")
	}
}

func TestServiceAccountApplicationKeyParseFilterID(t *testing.T) {
	tests := []struct {
		input             string
		wantServiceAcctID string
		wantKeyID         string
	}{
		{"sa-123:key-456", "sa-123", "key-456"},
		{"key-only", "", "key-only"},
		{"sa:key:extra", "sa", "key:extra"},
	}

	for _, tt := range tests {
		saID, keyID := parseServiceAccountApplicationKeyFilterID(tt.input)
		if saID != tt.wantServiceAcctID {
			t.Errorf("parseServiceAccountApplicationKeyFilterID(%q) serviceAccountID = %q, want %q", tt.input, saID, tt.wantServiceAcctID)
		}
		if keyID != tt.wantKeyID {
			t.Errorf("parseServiceAccountApplicationKeyFilterID(%q) keyID = %q, want %q", tt.input, keyID, tt.wantKeyID)
		}
	}
}

func TestServiceAccountApplicationKeyInitResourcesCompositeIDFilter(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newServiceAccountApplicationKeyTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "service_account_application_key",
			FieldPath:        "id",
			AcceptableValues: []string{"sa-123:key-456"},
		},
	})

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "key-456" {
		t.Fatalf("expected resource ID key-456, got %s", generator.Resources[0].InstanceState.ID)
	}
	if len(requestedPaths) != 0 {
		t.Fatalf("expected no API requests for ID filter, got %v", requestedPaths)
	}
}

func TestServiceAccountApplicationKeyInitResourcesServiceAccountIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/api/v2/service_accounts/sa-123/application_keys") {
			_, _ = fmt.Fprint(w, serviceAccountAppKeyListResponseJSON("key-1", "key-2"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	generator := newServiceAccountApplicationKeyTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "service_account_application_key",
			FieldPath:        "service_account_id",
			AcceptableValues: []string{"sa-123"},
		},
	})

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "key-1" {
		t.Fatalf("expected first resource ID key-1, got %s", generator.Resources[0].InstanceState.ID)
	}
}

func TestServiceAccountApplicationKeyInitResourcesNoFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newServiceAccountApplicationKeyTestGenerator(server, nil)

	// Should not error; just logs a message and returns empty
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("expected 0 resources without filter, got %d", len(generator.Resources))
	}
}

func TestServiceAccountApplicationKeyInitResourcesListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "list failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newServiceAccountApplicationKeyTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "service_account_application_key",
			FieldPath:        "service_account_id",
			AcceptableValues: []string{"sa-123"},
		},
	})

	if err := generator.InitResources(); err == nil {
		t.Fatal("expected error from list, got nil")
	}
}

func newServiceAccountApplicationKeyTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *ServiceAccountApplicationKeyGenerator {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return &ServiceAccountApplicationKeyGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": datadog.NewAPIClient(config),
				},
				Filter: filter,
			},
		},
	}
}

func serviceAccountAppKeyListResponseJSON(ids ...string) string {
	keys := []string{}
	for _, id := range ids {
		keys = append(keys, fmt.Sprintf("{\"id\":%q,\"type\":\"application_keys\"}", id))
	}
	return fmt.Sprintf("{\"data\":[%s],\"meta\":{\"page\":{\"total_filtered_count\":%d}}}", strings.Join(keys, ","), len(ids))
}
