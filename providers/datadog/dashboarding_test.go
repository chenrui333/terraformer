// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestDashboardV2CreateResource(t *testing.T) {
	generator := &DashboardV2Generator{}
	resource := generator.createResource("abc-def-123")

	if resource.InstanceState.ID != "abc-def-123" {
		t.Fatalf("resource ID = %q, want abc-def-123", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--dashboard_v2_abc-def-123" {
		t.Fatalf("resource name = %q, want tfer--dashboard_v2_abc-def-123", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_dashboard_v2" {
		t.Fatalf("resource type = %q, want datadog_dashboard_v2", resource.InstanceInfo.Type)
	}
}

func TestDashboardInitResourcesListsDashboardsWithPagination(t *testing.T) {
	tests := []struct {
		name string
		init func(*httptest.Server) ([]terraformutils.Resource, error)
	}{
		{
			name: "dashboard",
			init: func(server *httptest.Server) ([]terraformutils.Resource, error) {
				generator := newDashboardTestGenerator(server, nil)
				err := generator.InitResources()
				return generator.Resources, err
			},
		},
		{
			name: "dashboard_json",
			init: func(server *httptest.Server) ([]terraformutils.Resource, error) {
				generator := newDashboardJSONTestGenerator(server, nil)
				err := generator.InitResources()
				return generator.Resources, err
			},
		},
		{
			name: "dashboard_v2",
			init: func(server *httptest.Server) ([]terraformutils.Resource, error) {
				generator := newDashboardV2TestGenerator(server, nil)
				err := generator.InitResources()
				return generator.Resources, err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestStarts := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != "/api/v1/dashboard" {
					http.NotFound(w, r)
					return
				}
				if got := r.URL.Query().Get("count"); got != "100" {
					http.Error(w, fmt.Sprintf("count = %q, want 100", got), http.StatusBadRequest)
					return
				}

				start := r.URL.Query().Get("start")
				requestStarts = append(requestStarts, start)
				switch start {
				case "":
					_, _ = fmt.Fprint(w, dashboardSummaryResponseJSON(dashboardSummaryIDs("dashboard", 0, 100)...))
				case "100":
					_, _ = fmt.Fprint(w, dashboardSummaryResponseJSON("dashboard-100"))
				default:
					http.Error(w, fmt.Sprintf("start = %q, want empty or 100", start), http.StatusBadRequest)
				}
			}))
			defer server.Close()

			resources, err := tt.init(server)
			if err != nil {
				t.Fatalf("InitResources returned error: %v", err)
			}
			if len(resources) != 101 {
				t.Fatalf("expected 101 resources, got %d", len(resources))
			}
			if got := strings.Join(requestStarts, ","); got != ",100" {
				t.Fatalf("request starts = %q, want ,100", got)
			}
			if resources[0].InstanceState.ID != "dashboard-0" || resources[100].InstanceState.ID != "dashboard-100" {
				t.Fatalf("unexpected resource IDs: %s, %s", resources[0].InstanceState.ID, resources[100].InstanceState.ID)
			}
		})
	}
}

func TestPowerpackCreateResource(t *testing.T) {
	powerpack := datadogV2.NewPowerpackDataWithDefaults()
	powerpack.SetId("powerpack-1")

	tests := []struct {
		name     string
		create   func(datadogV2.PowerpackData) (terraformutils.Resource, error)
		wantName string
		wantType string
	}{
		{
			name:     "powerpack",
			create:   (&PowerpackGenerator{}).createResource,
			wantName: "tfer--powerpack_powerpack-1",
			wantType: "datadog_powerpack",
		},
		{
			name:     "powerpack_v2",
			create:   (&PowerpackV2Generator{}).createResource,
			wantName: "tfer--powerpack_v2_powerpack-1",
			wantType: "datadog_powerpack_v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := tt.create(*powerpack)
			if err != nil {
				t.Fatalf("createResource returned error: %v", err)
			}
			if resource.InstanceState.ID != "powerpack-1" {
				t.Fatalf("resource ID = %q, want powerpack-1", resource.InstanceState.ID)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %s", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %s", resource.InstanceInfo.Type, tt.wantType)
			}
		})
	}
}

func TestPowerpackAllowEmptyValuesPreservesManageStatusQuery(t *testing.T) {
	tests := []struct {
		name  string
		allow []string
	}{
		{
			name:  "powerpack",
			allow: PowerpackAllowEmptyValues,
		},
		{
			name:  "powerpack_v2",
			allow: PowerpackV2AllowEmptyValues,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, pattern := range tt.allow {
				if pattern == "manage_status_definition.*.query" {
					return
				}
			}

			t.Fatal("allow empty values must preserve manage_status_definition.*.query")
		})
	}
}

func TestPowerpackCreateResourceMissingID(t *testing.T) {
	if _, err := (&PowerpackGenerator{}).createResource(datadogV2.PowerpackData{}); err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
	if _, err := (&PowerpackV2Generator{}).createResource(datadogV2.PowerpackData{}); err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestPowerpackIDUsesUnparsedObject(t *testing.T) {
	powerpack := datadogV2.PowerpackData{
		UnparsedObject: map[string]interface{}{
			"id": "raw-powerpack-id",
		},
	}

	if got := powerpackID(powerpack); got != "raw-powerpack-id" {
		t.Fatalf("powerpackID = %q, want raw-powerpack-id", got)
	}
}

func TestPowerpackInitResourcesListsPowerpacks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/powerpacks" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("page[limit]"); got != "100" {
			http.Error(w, fmt.Sprintf("page[limit] = %q, want 100", got), http.StatusBadRequest)
			return
		}
		_, _ = fmt.Fprint(w, "{\"data\":[{\"id\":\"powerpack-1\",\"type\":\"powerpack\"},{\"id\":\"powerpack-2\",\"type\":\"powerpack\"}]}")
	}))
	defer server.Close()

	generator := newPowerpackTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "powerpack-1" || generator.Resources[1].InstanceState.ID != "powerpack-2" {
		t.Fatalf("unexpected resource IDs: %s, %s", generator.Resources[0].InstanceState.ID, generator.Resources[1].InstanceState.ID)
	}
}

func TestPowerpackV2InitResourcesFiltersByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/powerpacks/powerpack-1" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, "{\"data\":{\"id\":\"powerpack-1\",\"type\":\"powerpack\"}}")
	}))
	defer server.Close()

	generator := newPowerpackV2TestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "powerpack_v2",
			FieldPath:        "id",
			AcceptableValues: []string{"powerpack-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "powerpack-1" {
		t.Fatalf("resource ID = %q, want powerpack-1", generator.Resources[0].InstanceState.ID)
	}
}

func TestPowerpackInitResourcesSkipsListForSiblingIDFilter(t *testing.T) {
	tests := []struct {
		name          string
		filterService string
		init          func(*httptest.Server, []terraformutils.ResourceFilter) ([]terraformutils.Resource, error)
	}{
		{
			name:          "powerpack",
			filterService: "powerpack_v2",
			init: func(server *httptest.Server, filters []terraformutils.ResourceFilter) ([]terraformutils.Resource, error) {
				generator := newPowerpackTestGenerator(server, filters)
				err := generator.InitResources()
				return generator.Resources, err
			},
		},
		{
			name:          "powerpack_v2",
			filterService: "powerpack",
			init: func(server *httptest.Server, filters []terraformutils.ResourceFilter) ([]terraformutils.Resource, error) {
				generator := newPowerpackV2TestGenerator(server, filters)
				err := generator.InitResources()
				return generator.Resources, err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			}))
			defer server.Close()

			resources, err := tt.init(server, []terraformutils.ResourceFilter{
				{
					ServiceName:      tt.filterService,
					FieldPath:        "id",
					AcceptableValues: []string{"powerpack-1"},
				},
			})
			if err != nil {
				t.Fatalf("InitResources returned error: %v", err)
			}
			if len(resources) != 0 {
				t.Fatalf("expected 0 resources, got %d", len(resources))
			}
			if requests != 0 {
				t.Fatalf("expected no API requests, got %d", requests)
			}
		})
	}
}

func newDashboardTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *DashboardGenerator {
	return &DashboardGenerator{DatadogService: newDashboardingTestService(server, filter)}
}

func newDashboardJSONTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *DashboardJSONGenerator {
	return &DashboardJSONGenerator{DatadogService: newDashboardingTestService(server, filter)}
}

func newDashboardV2TestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *DashboardV2Generator {
	return &DashboardV2Generator{DatadogService: newDashboardingTestService(server, filter)}
}

func newPowerpackTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *PowerpackGenerator {
	return &PowerpackGenerator{DatadogService: newDashboardingTestService(server, filter)}
}

func newPowerpackV2TestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *PowerpackV2Generator {
	return &PowerpackV2Generator{DatadogService: newDashboardingTestService(server, filter)}
}

func newDashboardingTestService(server *httptest.Server, filter []terraformutils.ResourceFilter) DatadogService {
	return DatadogService{
		Service: terraformutils.Service{
			Args: map[string]interface{}{
				"auth":          context.Background(),
				"datadogClient": newTeamRelationshipTestClient(server),
			},
			Filter: filter,
		},
	}
}

func dashboardSummaryIDs(prefix string, start, count int) []string {
	ids := make([]string, 0, count)
	for i := start; i < start+count; i++ {
		ids = append(ids, fmt.Sprintf("%s-%d", prefix, i))
	}
	return ids
}

func dashboardSummaryResponseJSON(ids ...string) string {
	dashboards := make([]string, 0, len(ids))
	for _, id := range ids {
		dashboards = append(dashboards, fmt.Sprintf("{\"id\":%q}", id))
	}
	return fmt.Sprintf("{\"dashboards\":[%s]}", strings.Join(dashboards, ","))
}
