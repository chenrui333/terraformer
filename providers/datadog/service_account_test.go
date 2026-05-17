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

func TestServiceAccountCreateResource(t *testing.T) {
	generator := &ServiceAccountGenerator{}
	resource, err := generator.createResource("sa-123")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "sa-123" {
		t.Fatalf("expected resource ID sa-123, got %s", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != "datadog_service_account" {
		t.Fatalf("expected resource type datadog_service_account, got %s", resource.InstanceInfo.Type)
	}
}

func TestServiceAccountCreateResourceEmptyID(t *testing.T) {
	generator := &ServiceAccountGenerator{}
	_, err := generator.createResource("")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
}

func TestServiceAccountInitResourcesIDFilter(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newServiceAccountTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "service_account",
			FieldPath:        "id",
			AcceptableValues: []string{"sa-abc"},
		},
	})

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "sa-abc" {
		t.Fatalf("expected resource ID sa-abc, got %s", generator.Resources[0].InstanceState.ID)
	}
	if len(requestedPaths) != 0 {
		t.Fatalf("expected no API requests for ID filter, got %v", requestedPaths)
	}
}

func TestServiceAccountInitResourcesList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, serviceAccountListResponseJSON("sa-1", "sa-2"))
	}))
	defer server.Close()

	generator := newServiceAccountTestGenerator(server, nil)

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "sa-1" {
		t.Fatalf("expected first resource ID sa-1, got %s", generator.Resources[0].InstanceState.ID)
	}
}

func TestServiceAccountInitResourcesListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "list failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newServiceAccountTestGenerator(server, nil)

	if err := generator.InitResources(); err == nil {
		t.Fatal("expected error from list, got nil")
	}
}

func newServiceAccountTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *ServiceAccountGenerator {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return &ServiceAccountGenerator{
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

func serviceAccountListResponseJSON(ids ...string) string {
	users := []string{}
	for _, id := range ids {
		users = append(users, fmt.Sprintf("{\"id\":%q,\"type\":\"users\",\"attributes\":{\"service_account\":true}}", id))
	}
	return fmt.Sprintf("{\"data\":[%s],\"meta\":{\"page\":{\"total_count\":%d}}}", strings.Join(users, ","), len(ids))
}
