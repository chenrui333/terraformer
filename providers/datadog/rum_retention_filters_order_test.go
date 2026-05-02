// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestRumRetentionFiltersOrderCreateResource(t *testing.T) {
	firstFilter := datadogV2.NewRumRetentionFilterDataWithDefaults()
	firstFilter.SetId("rf-123")
	secondFilter := datadogV2.NewRumRetentionFilterDataWithDefaults()
	secondFilter.SetId("rf-456")

	generator := &RumRetentionFiltersOrderGenerator{}
	resource, err := generator.createResource("app-123", []datadogV2.RumRetentionFilterData{*firstFilter, *secondFilter})
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "app-123" {
		t.Fatalf("expected resource ID app-123, got %s", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--rum_retention_filters_order_app-123" {
		t.Fatalf("expected resource name tfer--rum_retention_filters_order_app-123, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_rum_retention_filters_order" {
		t.Fatalf("expected resource type datadog_rum_retention_filters_order, got %s", resource.InstanceInfo.Type)
	}
	if resource.InstanceState.Attributes["application_id"] != "app-123" {
		t.Fatalf("application_id = %q, want app-123", resource.InstanceState.Attributes["application_id"])
	}
	if resource.InstanceState.Attributes["retention_filter_ids.#"] != "2" {
		t.Fatalf("retention_filter_ids.# = %q, want 2", resource.InstanceState.Attributes["retention_filter_ids.#"])
	}
	if resource.InstanceState.Attributes["retention_filter_ids.0"] != "rf-123" {
		t.Fatalf("retention_filter_ids.0 = %q, want rf-123", resource.InstanceState.Attributes["retention_filter_ids.0"])
	}
	if resource.InstanceState.Attributes["retention_filter_ids.1"] != "rf-456" {
		t.Fatalf("retention_filter_ids.1 = %q, want rf-456", resource.InstanceState.Attributes["retention_filter_ids.1"])
	}
}

func TestRumRetentionFiltersOrderCreateResourceRequiresFilterIDs(t *testing.T) {
	filter := datadogV2.NewRumRetentionFilterDataWithDefaults()

	generator := &RumRetentionFiltersOrderGenerator{}
	if _, err := generator.createResource("", nil); err == nil {
		t.Fatal("createResource returned nil error, want missing application ID error")
	}
	if _, err := generator.createResource("app-123", []datadogV2.RumRetentionFilterData{*filter}); err == nil {
		t.Fatal("createResource returned nil error, want missing retention filter ID error")
	}
}

func TestRumRetentionFiltersOrderPostConvertHookPreservesEmptyRetentionFilterIDs(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"app-123",
		"rum_retention_filters_order_app-123",
		"datadog_rum_retention_filters_order",
		"datadog",
		RumRetentionFiltersOrderAllowEmptyValues,
	)
	resource.InstanceState.Attributes = map[string]string{
		"application_id":         "app-123",
		"retention_filter_ids.#": "0",
	}
	resource.Item = map[string]interface{}{
		"application_id": "app-123",
	}

	generator := &RumRetentionFiltersOrderGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	retentionFilterIDs, ok := generator.Resources[0].Item["retention_filter_ids"].([]interface{})
	if !ok {
		t.Fatalf("retention_filter_ids = %T, want []interface{}", generator.Resources[0].Item["retention_filter_ids"])
	}
	if len(retentionFilterIDs) != 0 {
		t.Fatalf("retention_filter_ids length = %d, want %d", len(retentionFilterIDs), 0)
	}
}

func TestRumRetentionFiltersOrderInitResourcesListsApplicationsAndFilters(t *testing.T) {
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

	generator := newRumRetentionFiltersOrderTestGenerator(server, nil)

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "app-123" {
		t.Fatalf("expected first resource ID app-123, got %s", generator.Resources[0].InstanceState.ID)
	}
	if generator.Resources[1].InstanceState.ID != "app-456" {
		t.Fatalf("expected second resource ID app-456, got %s", generator.Resources[1].InstanceState.ID)
	}
}

func TestRumRetentionFiltersOrderInitResourcesUsesApplicationIDFilter(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/rum/applications/app-123/retention_filters":
			_, _ = fmt.Fprint(w, rumRetentionFilterListResponseJSON("rf-123"))
		case "/api/v2/rum/applications":
			http.Error(w, "unexpected application list request", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	generator := newRumRetentionFiltersOrderTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "rum_retention_filters_order",
			FieldPath:        "application_id",
			AcceptableValues: []string{"app-123"},
		},
	})

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "app-123" {
		t.Fatalf("expected resource ID app-123, got %s", generator.Resources[0].InstanceState.ID)
	}
	for _, path := range requestedPaths {
		if path == "/api/v2/rum/applications" {
			t.Fatalf("expected application_id filter to avoid application list, got requests %v", requestedPaths)
		}
	}
}

func newRumRetentionFiltersOrderTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *RumRetentionFiltersOrderGenerator {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return &RumRetentionFiltersOrderGenerator{
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
