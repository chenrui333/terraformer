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

func TestApplicationKeyCreateResource(t *testing.T) {
	generator := &ApplicationKeyGenerator{}
	resource, err := generator.createResource("appkey-123")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "appkey-123" {
		t.Fatalf("expected resource ID appkey-123, got %s", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != "datadog_application_key" {
		t.Fatalf("expected resource type datadog_application_key, got %s", resource.InstanceInfo.Type)
	}
}

func TestApplicationKeyCreateResourceEmptyID(t *testing.T) {
	generator := &ApplicationKeyGenerator{}
	_, err := generator.createResource("")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
}

func TestApplicationKeyInitResourcesIDFilter(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newApplicationKeyTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "application_key",
			FieldPath:        "id",
			AcceptableValues: []string{"appkey-abc"},
		},
	})

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "appkey-abc" {
		t.Fatalf("expected resource ID appkey-abc, got %s", generator.Resources[0].InstanceState.ID)
	}
	if len(requestedPaths) != 0 {
		t.Fatalf("expected no API requests for ID filter, got %v", requestedPaths)
	}
}

func TestApplicationKeyInitResourcesList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, applicationKeyListResponseJSON("appkey-1", "appkey-2"))
	}))
	defer server.Close()

	generator := newApplicationKeyTestGenerator(server, nil)

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "appkey-1" {
		t.Fatalf("expected first resource ID appkey-1, got %s", generator.Resources[0].InstanceState.ID)
	}
}

func TestApplicationKeyInitResourcesListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "list failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newApplicationKeyTestGenerator(server, nil)

	if err := generator.InitResources(); err == nil {
		t.Fatal("expected error from list, got nil")
	}
}

func newApplicationKeyTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *ApplicationKeyGenerator {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return &ApplicationKeyGenerator{
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

func applicationKeyListResponseJSON(ids ...string) string {
	keys := []string{}
	for _, id := range ids {
		keys = append(keys, fmt.Sprintf("{\"id\":%q,\"type\":\"application_keys\"}", id))
	}
	return fmt.Sprintf("{\"data\":[%s],\"meta\":{\"page\":{\"total_filtered_count\":%d}}}", strings.Join(keys, ","), len(ids))
}
