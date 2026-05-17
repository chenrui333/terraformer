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

func TestAuthnMappingCreateResource(t *testing.T) {
	generator := &AuthnMappingGenerator{}
	resource, err := generator.createResource("mapping-123")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "mapping-123" {
		t.Fatalf("expected resource ID mapping-123, got %s", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != "datadog_authn_mapping" {
		t.Fatalf("expected resource type datadog_authn_mapping, got %s", resource.InstanceInfo.Type)
	}
}

func TestAuthnMappingCreateResourceEmptyID(t *testing.T) {
	generator := &AuthnMappingGenerator{}
	_, err := generator.createResource("")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
}

func TestAuthnMappingInitResourcesIDFilter(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newAuthnMappingTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "authn_mapping",
			FieldPath:        "id",
			AcceptableValues: []string{"mapping-abc"},
		},
	})

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "mapping-abc" {
		t.Fatalf("expected resource ID mapping-abc, got %s", generator.Resources[0].InstanceState.ID)
	}
	if len(requestedPaths) != 0 {
		t.Fatalf("expected no API requests for ID filter, got %v", requestedPaths)
	}
}

func TestAuthnMappingInitResourcesList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, authnMappingListResponseJSON("mapping-1", "mapping-2"))
	}))
	defer server.Close()

	generator := newAuthnMappingTestGenerator(server, nil)

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "mapping-1" {
		t.Fatalf("expected first resource ID mapping-1, got %s", generator.Resources[0].InstanceState.ID)
	}
	if generator.Resources[1].InstanceState.ID != "mapping-2" {
		t.Fatalf("expected second resource ID mapping-2, got %s", generator.Resources[1].InstanceState.ID)
	}
}

func TestAuthnMappingInitResourcesListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "list failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newAuthnMappingTestGenerator(server, nil)

	if err := generator.InitResources(); err == nil {
		t.Fatal("expected error from list, got nil")
	}
}

func newAuthnMappingTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *AuthnMappingGenerator {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return &AuthnMappingGenerator{
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

func authnMappingListResponseJSON(ids ...string) string {
	mappings := []string{}
	for _, id := range ids {
		mappings = append(mappings, fmt.Sprintf("{\"id\":%q,\"type\":\"authn_mappings\"}", id))
	}
	return fmt.Sprintf("{\"data\":[%s],\"meta\":{\"page\":{\"total_count\":%d}}}", strings.Join(mappings, ","), len(ids))
}
