// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestObservabilityPipelineCreateResource(t *testing.T) {
	resource, err := (&ObservabilityPipelineGenerator{}).createResource(observabilityPipelineData("pipeline-123"))
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}
	if resource.InstanceState.ID != "pipeline-123" {
		t.Fatalf("resource ID = %q, want pipeline-123", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--observability_pipeline_pipeline-123" {
		t.Fatalf("resource name = %q, want tfer--observability_pipeline_pipeline-123", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_observability_pipeline" {
		t.Fatalf("resource type = %q, want datadog_observability_pipeline", resource.InstanceInfo.Type)
	}
}

func TestObservabilityPipelineCreateResourceMissingID(t *testing.T) {
	if _, err := (&ObservabilityPipelineGenerator{}).createResource(observabilityPipelineData("")); err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestObservabilityPipelineAllowEmptyValuesPreservesFalseBooleans(t *testing.T) {
	requiredPaths := []string{
		"config.processor_group.enabled",
		"config.processor_group.processor.enabled",
		"config.processor_group.processor.custom_processor.remap.drop_on_error",
		"config.processor_group.processor.custom_processor.remap.enabled",
		"config.processor_group.processor.rename_fields.field.preserve_source",
	}
	for _, path := range requiredPaths {
		if !slices.Contains(ObservabilityPipelineAllowEmptyValues, path) {
			t.Fatalf("ObservabilityPipelineAllowEmptyValues must include %q", path)
		}
	}
}

func TestObservabilityPipelineInitResourcesListsPipelinesWithPagination(t *testing.T) {
	requestPages := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/obs-pipelines/pipelines" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("page[size]"); got != "100" {
			http.Error(w, fmt.Sprintf("page[size] = %q, want 100", got), http.StatusBadRequest)
			return
		}

		pageNumber := r.URL.Query().Get("page[number]")
		requestPages = append(requestPages, pageNumber)
		switch pageNumber {
		case "0":
			_, _ = fmt.Fprint(w, observabilityPipelineListResponseJSON(101, observabilityPipelineIDs("pipeline", 0, 100)...))
		case "1":
			_, _ = fmt.Fprint(w, observabilityPipelineListResponseJSON(101, "pipeline-100"))
		default:
			http.Error(w, fmt.Sprintf("page[number] = %q, want 0 or 1", pageNumber), http.StatusBadRequest)
		}
	}))
	defer server.Close()

	generator := newObservabilityPipelineTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if strings.Join(requestPages, ",") != "0,1" {
		t.Fatalf("request pages = %v, want [0 1]", requestPages)
	}
	assertResourceIDs(t, generator.Resources, observabilityPipelineIDs("pipeline", 0, 101))
}

func TestObservabilityPipelineInitResourcesHandlesEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/obs-pipelines/pipelines" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, observabilityPipelineListResponseJSON(0))
	}))
	defer server.Close()

	generator := newObservabilityPipelineTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("resources length = %d, want 0", len(generator.Resources))
	}
}

func TestObservabilityPipelineInitResourcesReturnsForbiddenListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/obs-pipelines/pipelines" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	generator := newObservabilityPipelineTestGenerator(server, nil)
	if err := generator.InitResources(); err == nil {
		t.Fatal("InitResources returned nil error, want forbidden error")
	}
}

func TestObservabilityPipelineInitResourcesFiltersByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/obs-pipelines/pipelines/pipeline-123" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, observabilityPipelineGetResponseJSON("pipeline-123"))
	}))
	defer server.Close()

	generator := newObservabilityPipelineTestGenerator(server, []terraformutils.ResourceFilter{{
		ServiceName:      "observability_pipeline",
		FieldPath:        "id",
		AcceptableValues: []string{"pipeline-123"},
	}})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"pipeline-123"})
}

func TestObservabilityPipelineInitResourcesSuppressesBroadDiscoveryForUnrelatedIDFilter(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.NotFound(w, r)
	}))
	defer server.Close()

	generator := newObservabilityPipelineTestGenerator(server, []terraformutils.ResourceFilter{{
		ServiceName:      "logs_custom_destination",
		FieldPath:        "id",
		AcceptableValues: []string{"destination-id"},
	}})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("resources length = %d, want 0", len(generator.Resources))
	}
}

func newObservabilityPipelineTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *ObservabilityPipelineGenerator {
	return &ObservabilityPipelineGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": newTeamRelationshipTestClient(server),
				},
				Filter: filter,
			},
		},
	}
}

func observabilityPipelineIDs(prefix string, start, count int) []string {
	ids := make([]string, 0, count)
	for i := start; i < start+count; i++ {
		ids = append(ids, fmt.Sprintf("%s-%d", prefix, i))
	}
	return ids
}

func observabilityPipelineGetResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":%s}", observabilityPipelineJSON(id))
}

func observabilityPipelineListResponseJSON(totalCount int, ids ...string) string {
	items := make([]string, 0, len(ids))
	for _, id := range ids {
		items = append(items, observabilityPipelineJSON(id))
	}
	return fmt.Sprintf("{\"data\":[%s],\"meta\":{\"totalCount\":%d}}", strings.Join(items, ","), totalCount)
}

func observabilityPipelineJSON(id string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"pipelines\",\"attributes\":{\"name\":%q,\"config\":{\"sources\":[],\"destinations\":[]}}}",
		id,
		fmt.Sprintf("pipeline %s", id),
	)
}

func observabilityPipelineData(id string) datadogV2.ObservabilityPipelineData {
	return *datadogV2.NewObservabilityPipelineData(
		*datadogV2.NewObservabilityPipelineDataAttributes(
			*datadogV2.NewObservabilityPipelineConfig([]datadogV2.ObservabilityPipelineConfigDestinationItem{}, []datadogV2.ObservabilityPipelineConfigSourceItem{}),
			fmt.Sprintf("pipeline %s", id),
		),
		id,
		"pipelines",
	)
}
