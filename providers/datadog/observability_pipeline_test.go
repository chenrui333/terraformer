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

func TestObservabilityPipelinePostConvertHookPreservesEmptyVariantBlocks(t *testing.T) {
	resource := observabilityPipelineResourceWithItem(map[string]interface{}{
		"config": []interface{}{
			map[string]interface{}{
				"destination": []interface{}{
					map[string]interface{}{"id": "logs", "inputs": []interface{}{"source"}},
					map[string]interface{}{"id": "metrics", "inputs": []interface{}{"source"}},
				},
				"processor_group": []interface{}{
					map[string]interface{}{
						"processor": []interface{}{
							map[string]interface{}{"id": "filter"},
							map[string]interface{}{"id": "hostname"},
							map[string]interface{}{
								"id": "scanner",
								"sensitive_data_scanner": []interface{}{
									map[string]interface{}{
										"rule": []interface{}{
											map[string]interface{}{
												"on_match": []interface{}{
													map[string]interface{}{},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	resource.InstanceState.Attributes = map[string]string{
		"config.0.destination.0.datadog_logs.#":                                                    "1",
		"config.0.destination.1.datadog_metrics.#":                                                 "1",
		"config.0.processor_group.0.processor.0.filter.#":                                          "1",
		"config.0.processor_group.0.processor.1.add_hostname.#":                                    "1",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.0.on_match.0.hash.#": "1",
	}
	generator := &ObservabilityPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	config := requireMapInList(t, generator.Resources[0].Item, "config", 0)
	requireEmptyBlockList(t, requireMapInList(t, config, "destination", 0), "datadog_logs")
	requireEmptyBlockList(t, requireMapInList(t, config, "destination", 1), "datadog_metrics")

	processorGroup := requireMapInList(t, config, "processor_group", 0)
	requireEmptyBlockList(t, requireMapInList(t, processorGroup, "processor", 0), "filter")
	requireEmptyBlockList(t, requireMapInList(t, processorGroup, "processor", 1), "add_hostname")

	scannerProcessor := requireMapInList(t, processorGroup, "processor", 2)
	scanner := requireMapInList(t, scannerProcessor, "sensitive_data_scanner", 0)
	rule := requireMapInList(t, scanner, "rule", 0)
	onMatch := requireMapInList(t, rule, "on_match", 0)
	requireEmptyBlockList(t, onMatch, "hash")
}

func TestObservabilityPipelinePostConvertHookDoesNotInventMissingVariantBlocks(t *testing.T) {
	resource := observabilityPipelineResourceWithItem(map[string]interface{}{
		"config": []interface{}{
			map[string]interface{}{
				"destination": []interface{}{
					map[string]interface{}{"id": "logs", "inputs": []interface{}{"source"}},
				},
			},
		},
	})
	resource.InstanceState.Attributes = map[string]string{"id": "pipeline-123"}
	generator := &ObservabilityPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	config := requireMapInList(t, generator.Resources[0].Item, "config", 0)
	destination := requireMapInList(t, config, "destination", 0)
	if _, ok := destination["datadog_logs"]; ok {
		t.Fatal("PostConvertHook added datadog_logs without a matching state count marker")
	}
}

func TestObservabilityPipelinePostConvertHookDoesNotOverwriteExistingVariantBlocks(t *testing.T) {
	existing := []interface{}{
		map[string]interface{}{"routes": []interface{}{"route-a"}},
	}
	resource := observabilityPipelineResourceWithItem(map[string]interface{}{
		"config": []interface{}{
			map[string]interface{}{
				"destination": []interface{}{
					map[string]interface{}{
						"id":           "logs",
						"inputs":       []interface{}{"source"},
						"datadog_logs": existing,
					},
				},
			},
		},
	})
	resource.InstanceState.Attributes = map[string]string{
		"config.0.destination.0.datadog_logs.#": "1",
	}
	generator := &ObservabilityPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	config := requireMapInList(t, generator.Resources[0].Item, "config", 0)
	destination := requireMapInList(t, config, "destination", 0)
	got := requireList(t, destination, "datadog_logs")
	if len(got) != 1 {
		t.Fatalf("datadog_logs length = %d, want 1", len(got))
	}
	block, ok := got[0].(map[string]interface{})
	if !ok {
		t.Fatalf("datadog_logs[0] = %T, want map[string]interface{}", got[0])
	}
	if _, ok := block["routes"]; !ok {
		t.Fatalf("datadog_logs = %#v, want existing routes preserved", got)
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

func TestObservabilityPipelineInitResourcesListsWithUnrelatedIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/obs-pipelines/pipelines" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, observabilityPipelineListResponseJSON(1, "pipeline-123"))
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
	assertResourceIDs(t, generator.Resources, []string{"pipeline-123"})
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

func observabilityPipelineResourceWithItem(item map[string]interface{}) terraformutils.Resource {
	resource := terraformutils.NewSimpleResource(
		"pipeline-123",
		"observability_pipeline_pipeline-123",
		"datadog_observability_pipeline",
		"datadog",
		ObservabilityPipelineAllowEmptyValues,
	)
	resource.Item = item
	return resource
}

func requireList(t *testing.T, parent map[string]interface{}, key string) []interface{} {
	t.Helper()
	list, ok := parent[key].([]interface{})
	if !ok {
		t.Fatalf("%s = %T, want []interface{}", key, parent[key])
	}
	return list
}

func requireMapInList(t *testing.T, parent map[string]interface{}, key string, index int) map[string]interface{} {
	t.Helper()
	list := requireList(t, parent, key)
	if len(list) <= index {
		t.Fatalf("%s length = %d, want index %d", key, len(list), index)
	}
	item, ok := list[index].(map[string]interface{})
	if !ok {
		t.Fatalf("%s[%d] = %T, want map[string]interface{}", key, index, list[index])
	}
	return item
}

func requireEmptyBlockList(t *testing.T, parent map[string]interface{}, key string) {
	t.Helper()
	list := requireList(t, parent, key)
	if len(list) != 1 {
		t.Fatalf("%s length = %d, want 1", key, len(list))
	}
	block, ok := list[0].(map[string]interface{})
	if !ok {
		t.Fatalf("%s[0] = %T, want map[string]interface{}", key, list[0])
	}
	if len(block) != 0 {
		t.Fatalf("%s[0] = %#v, want empty block", key, block)
	}
}
