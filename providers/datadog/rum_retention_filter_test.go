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
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestRumRetentionFilterCreateResource(t *testing.T) {
	filter := datadogV2.NewRumRetentionFilterDataWithDefaults()
	filter.SetId("rf-123")

	generator := &RumRetentionFilterGenerator{}
	resource, err := generator.createResource("app-123", *filter)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "rf-123" {
		t.Fatalf("expected resource ID rf-123, got %s", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--rum_retention_filter_app-123_rf-123" {
		t.Fatalf("expected resource name tfer--rum_retention_filter_app-123_rf-123, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_rum_retention_filter" {
		t.Fatalf("expected resource type datadog_rum_retention_filter, got %s", resource.InstanceInfo.Type)
	}
	if resource.InstanceState.Attributes["application_id"] != "app-123" {
		t.Fatalf("application_id = %q, want app-123", resource.InstanceState.Attributes["application_id"])
	}
}

func TestRumRetentionFilterCreateResourceRequiresIDs(t *testing.T) {
	filter := datadogV2.NewRumRetentionFilterDataWithDefaults()

	generator := &RumRetentionFilterGenerator{}
	if _, err := generator.createResource("app-123", *filter); err == nil {
		t.Fatal("createResource returned nil error, want missing retention filter ID error")
	}

	filter.SetId("rf-123")
	if _, err := generator.createResource("", *filter); err == nil {
		t.Fatal("createResource returned nil error, want missing application ID error")
	}
}

func TestRumRetentionFilterParseImportID(t *testing.T) {
	applicationID, retentionFilterID, err := parseRumRetentionFilterImportID("app-123:rf-123")
	if err != nil {
		t.Fatalf("parseRumRetentionFilterImportID returned error: %v", err)
	}
	if applicationID != "app-123" {
		t.Fatalf("applicationID = %q, want app-123", applicationID)
	}
	if retentionFilterID != "rf-123" {
		t.Fatalf("retentionFilterID = %q, want rf-123", retentionFilterID)
	}
}

func TestRumRetentionFilterParseImportIDRejectsInvalid(t *testing.T) {
	if _, _, err := parseRumRetentionFilterImportID("rf-123"); err == nil {
		t.Fatal("parseRumRetentionFilterImportID returned nil error, want invalid format error")
	}
}

func TestRumRetentionFilterInitResourcesFetchesIDFilter(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/rum/applications/app-123/retention_filters/rf-123":
			_, _ = fmt.Fprint(w, rumRetentionFilterResponseJSON("rf-123"))
		case "/api/v2/rum/applications/app-123/retention_filters", "/api/v2/rum/applications":
			http.Error(w, "unexpected list request", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	generator := newRumRetentionFilterTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "rum_retention_filter",
			FieldPath:        "id",
			AcceptableValues: []string{"app-123:rf-123"},
		},
	})

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "rf-123" {
		t.Fatalf("expected resource ID rf-123, got %s", generator.Resources[0].InstanceState.ID)
	}
	if got := generator.Filter[0].AcceptableValues; len(got) != 1 || got[0] != "rf-123" {
		t.Fatalf("expected normalized filter ID rf-123, got %v", got)
	}
	for _, path := range requestedPaths {
		if path == "/api/v2/rum/applications" || path == "/api/v2/rum/applications/app-123/retention_filters" {
			t.Fatalf("expected id filter to avoid list requests, got requests %v", requestedPaths)
		}
	}
}

func TestRumRetentionFilterInitResourcesListsApplicationsAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/rum/applications":
			_, _ = fmt.Fprint(w, rumApplicationListResponseJSON("app-123", "app-456"))
		case "/api/v2/rum/applications/app-123/retention_filters":
			_, _ = fmt.Fprint(w, rumRetentionFilterListResponseJSON("rf-123"))
		case "/api/v2/rum/applications/app-456/retention_filters":
			_, _ = fmt.Fprint(w, rumRetentionFilterListResponseJSON("rf-456"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	generator := newRumRetentionFilterTestGenerator(server, nil)

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "rf-123" {
		t.Fatalf("expected first resource ID rf-123, got %s", generator.Resources[0].InstanceState.ID)
	}
	if generator.Resources[1].InstanceState.ID != "rf-456" {
		t.Fatalf("expected second resource ID rf-456, got %s", generator.Resources[1].InstanceState.ID)
	}
}

func TestRumRetentionFilterInitResourcesPropagatesListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/rum/applications":
			_, _ = fmt.Fprint(w, rumApplicationListResponseJSON("app-123"))
		case "/api/v2/rum/applications/app-123/retention_filters":
			http.Error(w, "list failed", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	generator := newRumRetentionFilterTestGenerator(server, nil)

	if err := generator.InitResources(); err == nil {
		t.Fatal("InitResources returned nil error, want list error")
	}
}

func newRumRetentionFilterTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *RumRetentionFilterGenerator {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return &RumRetentionFilterGenerator{
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

func rumRetentionFilterResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":%s}", rumRetentionFilterJSON(id))
}

func rumRetentionFilterListResponseJSON(ids ...string) string {
	filters := []string{}
	for _, id := range ids {
		filters = append(filters, rumRetentionFilterJSON(id))
	}
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(filters, ","))
}

func rumRetentionFilterJSON(id string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"retention_filters\",\"attributes\":{\"name\":\"Test filter\",\"event_type\":\"session\",\"sample_rate\":100,\"query\":\"\",\"enabled\":true}}",
		id,
	)
}
