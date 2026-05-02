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

func TestAPMRetentionFilterCreateResource(t *testing.T) {
	generator := &APMRetentionFilterGenerator{}
	resource := generator.createResource("rf-123")

	if resource.InstanceState.ID != "rf-123" {
		t.Fatalf("expected resource ID rf-123, got %s", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--apm_retention_filter_rf-123" {
		t.Fatalf("expected resource name tfer--apm_retention_filter_rf-123, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_apm_retention_filter" {
		t.Fatalf("expected resource type datadog_apm_retention_filter, got %s", resource.InstanceInfo.Type)
	}
}

func TestAPMRetentionFilterCreateResources(t *testing.T) {
	firstFilter := datadogV2.NewRetentionFilterAllWithDefaults()
	firstFilter.SetId("rf-123")
	firstFilter.SetAttributes(importableAPMRetentionFilterAttributes())

	secondFilter := datadogV2.NewRetentionFilterAllWithDefaults()
	secondFilter.SetId("rf-456")
	secondFilter.SetAttributes(importableAPMRetentionFilterAttributes())

	generator := &APMRetentionFilterGenerator{}
	resources := generator.createResources([]datadogV2.RetentionFilterAll{*firstFilter, *secondFilter})

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "rf-123" {
		t.Fatalf("expected first resource ID rf-123, got %s", resources[0].InstanceState.ID)
	}
	if resources[1].InstanceState.ID != "rf-456" {
		t.Fatalf("expected second resource ID rf-456, got %s", resources[1].InstanceState.ID)
	}
}

func TestAPMRetentionFilterCreateResourcesSkipsUnsupportedFilterTypes(t *testing.T) {
	userFilter := datadogV2.NewRetentionFilterAllWithDefaults()
	userFilter.SetId("rf-user")
	userFilter.SetAttributes(importableAPMRetentionFilterAttributes())

	defaultFilter := datadogV2.NewRetentionFilterAllWithDefaults()
	defaultFilter.SetId("rf-default")
	attributes := importableAPMRetentionFilterAttributes()
	attributes.SetFilterType(datadogV2.RETENTIONFILTERALLTYPE_SPANS_ERRORS_SAMPLING_PROCESSOR)
	defaultFilter.SetAttributes(attributes)

	generator := &APMRetentionFilterGenerator{}
	resources := generator.createResources([]datadogV2.RetentionFilterAll{*userFilter, *defaultFilter})

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "rf-user" {
		t.Fatalf("expected resource ID rf-user, got %s", resources[0].InstanceState.ID)
	}
}

func TestAPMRetentionFilterInitResourcesFetchesIDFilter(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/apm/config/retention-filters/rf-123":
			_, _ = fmt.Fprint(w, apmRetentionFilterResponseJSON("rf-123"))
		case "/api/v2/apm/config/retention-filters":
			http.Error(w, "unexpected list request", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	generator := newAPMRetentionFilterTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "apm_retention_filter",
			FieldPath:        "id",
			AcceptableValues: []string{"rf-123"},
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
	for _, path := range requestedPaths {
		if path == "/api/v2/apm/config/retention-filters" {
			t.Fatalf("expected id filter to avoid list request, got requests %v", requestedPaths)
		}
	}
}

func TestAPMRetentionFilterInitResourcesListsWithoutIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/apm/config/retention-filters":
			_, _ = fmt.Fprint(w, apmRetentionFilterListResponseJSON("rf-123", "rf-456"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	generator := newAPMRetentionFilterTestGenerator(server, nil)

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

func TestAPMRetentionFilterInitResourcesPropagatesIDFilterError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "fetch failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newAPMRetentionFilterTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "apm_retention_filter",
			FieldPath:        "id",
			AcceptableValues: []string{"rf-error"},
		},
	})

	if err := generator.InitResources(); err == nil {
		t.Fatal("InitResources returned nil error, want ID fetch error")
	}
}

func TestAPMRetentionFilterInitResourcesPropagatesListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "list failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newAPMRetentionFilterTestGenerator(server, nil)

	if err := generator.InitResources(); err == nil {
		t.Fatal("InitResources returned nil error, want list error")
	}
}

func importableAPMRetentionFilterAttributes() datadogV2.RetentionFilterAllAttributes {
	attributes := datadogV2.NewRetentionFilterAllAttributesWithDefaults()
	attributes.SetFilterType(datadogV2.RETENTIONFILTERALLTYPE_SPANS_SAMPLING_PROCESSOR)
	return *attributes
}

func newAPMRetentionFilterTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *APMRetentionFilterGenerator {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return &APMRetentionFilterGenerator{
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

func apmRetentionFilterResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":%s}", apmRetentionFilterJSON(id))
}

func apmRetentionFilterListResponseJSON(ids ...string) string {
	filters := []string{}
	for _, id := range ids {
		filters = append(filters, apmRetentionFilterJSON(id))
	}
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(filters, ","))
}

func apmRetentionFilterJSON(id string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"apm_retention_filter\",\"attributes\":{\"filter_type\":\"spans-sampling-processor\"}}",
		id,
	)
}
