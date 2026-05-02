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

func TestRumApplicationCreateResource(t *testing.T) {
	generator := &RumApplicationGenerator{}
	resource := generator.createResource("app-123")

	if resource.InstanceState.ID != "app-123" {
		t.Fatalf("expected resource ID app-123, got %s", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--rum_application_app-123" {
		t.Fatalf("expected resource name tfer--rum_application_app-123, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_rum_application" {
		t.Fatalf("expected resource type datadog_rum_application, got %s", resource.InstanceInfo.Type)
	}
}

func TestRumApplicationCreateResources(t *testing.T) {
	firstApplication := datadogV2.NewRUMApplicationListWithDefaults()
	firstApplication.SetId("app-123")

	secondApplication := datadogV2.NewRUMApplicationListWithDefaults()
	secondApplication.SetId("app-456")

	generator := &RumApplicationGenerator{}
	resources := generator.createResources([]datadogV2.RUMApplicationList{*firstApplication, *secondApplication})

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "app-123" {
		t.Fatalf("expected first resource ID app-123, got %s", resources[0].InstanceState.ID)
	}
	if resources[1].InstanceState.ID != "app-456" {
		t.Fatalf("expected second resource ID app-456, got %s", resources[1].InstanceState.ID)
	}
}

func TestRumApplicationCreateResourcesSkipsEmptyIDs(t *testing.T) {
	validApplication := datadogV2.NewRUMApplicationListWithDefaults()
	validApplication.SetId("app-123")

	missingIDApplication := datadogV2.NewRUMApplicationListWithDefaults()

	generator := &RumApplicationGenerator{}
	resources := generator.createResources([]datadogV2.RUMApplicationList{*missingIDApplication, *validApplication})

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "app-123" {
		t.Fatalf("expected resource ID app-123, got %s", resources[0].InstanceState.ID)
	}
}

func TestRumApplicationCreateResourcesFallsBackToAttributeApplicationID(t *testing.T) {
	application := datadogV2.NewRUMApplicationListWithDefaults()
	attributes := datadogV2.NewRUMApplicationListAttributesWithDefaults()
	attributes.SetApplicationId("app-attributes")
	application.SetAttributes(*attributes)

	generator := &RumApplicationGenerator{}
	resources := generator.createResources([]datadogV2.RUMApplicationList{*application})

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "app-attributes" {
		t.Fatalf("expected resource ID app-attributes, got %s", resources[0].InstanceState.ID)
	}
}

func TestRumApplicationInitResourcesFetchesIDFilter(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/rum/applications/app-123":
			_, _ = fmt.Fprint(w, rumApplicationResponseJSON("app-123"))
		case "/api/v2/rum/applications":
			http.Error(w, "unexpected list request", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	generator := newRumApplicationTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "rum_application",
			FieldPath:        "id",
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
			t.Fatalf("expected id filter to avoid list request, got requests %v", requestedPaths)
		}
	}
}

func TestRumApplicationInitResourcesListsWithoutIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/rum/applications":
			_, _ = fmt.Fprint(w, rumApplicationListResponseJSON("app-123", "app-456"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	generator := newRumApplicationTestGenerator(server, nil)

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

func TestRumApplicationInitResourcesPropagatesIDFilterError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "fetch failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newRumApplicationTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "rum_application",
			FieldPath:        "id",
			AcceptableValues: []string{"app-error"},
		},
	})

	if err := generator.InitResources(); err == nil {
		t.Fatal("InitResources returned nil error, want ID fetch error")
	}
}

func TestRumApplicationInitResourcesPropagatesListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "list failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	generator := newRumApplicationTestGenerator(server, nil)

	if err := generator.InitResources(); err == nil {
		t.Fatal("InitResources returned nil error, want list error")
	}
}

func newRumApplicationTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *RumApplicationGenerator {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return &RumApplicationGenerator{
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

func rumApplicationResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":%s}", rumApplicationJSON(id))
}

func rumApplicationListResponseJSON(ids ...string) string {
	applications := []string{}
	for _, id := range ids {
		applications = append(applications, rumApplicationJSON(id))
	}
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(applications, ","))
}

func rumApplicationJSON(id string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"rum_application\",\"attributes\":{\"application_id\":%q,\"created_at\":1700000000000,\"created_by_handle\":\"user@example.com\",\"name\":\"Test application\",\"org_id\":123,\"type\":\"browser\",\"updated_at\":1700000000000,\"updated_by_handle\":\"user@example.com\"}}",
		id,
		id,
	)
}
