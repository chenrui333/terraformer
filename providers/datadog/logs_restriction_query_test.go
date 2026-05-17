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

func TestLogsRestrictionQueryCreateResource(t *testing.T) {
	generator := &LogsRestrictionQueryGenerator{}
	resource, err := generator.createResource("query-123")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "query-123" {
		t.Fatalf("expected resource ID query-123, got %s", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != "datadog_logs_restriction_query" {
		t.Fatalf("expected resource type datadog_logs_restriction_query, got %s", resource.InstanceInfo.Type)
	}
}

func TestLogsRestrictionQueryCreateResourceEmptyID(t *testing.T) {
	generator := &LogsRestrictionQueryGenerator{}
	_, err := generator.createResource("")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
}

func TestLogsRestrictionQueryInitResourcesList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, logsRestrictionQueryListResponseJSON("query-1", "query-2"))
	}))
	defer server.Close()

	generator := newLogsRestrictionQueryTestGenerator(server, nil)

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "query-1" {
		t.Fatalf("expected first resource ID query-1, got %s", generator.Resources[0].InstanceState.ID)
	}
	if generator.Resources[1].InstanceState.ID != "query-2" {
		t.Fatalf("expected second resource ID query-2, got %s", generator.Resources[1].InstanceState.ID)
	}
}

func TestLogsRestrictionQueryInitResourcesListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "list failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newLogsRestrictionQueryTestGenerator(server, nil)

	if err := generator.InitResources(); err == nil {
		t.Fatal("expected error from list, got nil")
	}
}

func newLogsRestrictionQueryTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *LogsRestrictionQueryGenerator {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()
	config.SetUnstableOperationEnabled("v2.ListRestrictionQueries", true)

	return &LogsRestrictionQueryGenerator{
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

func logsRestrictionQueryListResponseJSON(ids ...string) string {
	queries := []string{}
	for _, id := range ids {
		queries = append(queries, fmt.Sprintf("{\"id\":%q,\"type\":\"logs_restriction_queries\"}", id))
	}
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(queries, ","))
}
