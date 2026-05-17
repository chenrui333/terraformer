// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"

	"github.com/chenrui333/terraformer/terraformutils"
)

type productPlatformGeneratorFactory func(*httptest.Server, []terraformutils.ResourceFilter) (func() error, func() []terraformutils.Resource)

func TestProductPlatformCreateResource(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		resourceType string
		create       func(string) (terraformutils.Resource, error)
	}{
		{
			name:         datadogAppBuilderAppServiceName,
			id:           "00000000-0000-0000-0000-000000000001",
			resourceType: "datadog_app_builder_app",
			create:       (&AppBuilderAppGenerator{}).createResource,
		},
		{name: datadogOpenapiAPIServiceName, id: "00000000-0000-0000-0000-000000000010", resourceType: "datadog_openapi_api", create: (&OpenapiAPIGenerator{}).createResource},
		{name: datadogDeploymentGateServiceName, id: "gate-1", resourceType: "datadog_deployment_gate", create: (&DeploymentGateGenerator{}).createResource},
		{name: datadogDatasetServiceName, id: "dataset-1", resourceType: "datadog_dataset", create: (&DatasetGenerator{}).createResource},
		{name: datadogDatastoreServiceName, id: "datastore-1", resourceType: "datadog_datastore", create: (&DatastoreGenerator{}).createResource},
		{name: datadogReferenceTableServiceName, id: "table-1", resourceType: "datadog_reference_table", create: (&ReferenceTableGenerator{}).createResource},
		{name: datadogObservabilityPipelineServiceName, id: "pipeline-1", resourceType: "datadog_observability_pipeline", create: (&ObservabilityPipelineGenerator{}).createResource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := tt.create(tt.id)
			if err != nil {
				t.Fatalf("createResource returned error: %v", err)
			}
			if resource.InstanceState.ID != tt.id {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.id)
			}
			if resource.ResourceName != fmt.Sprintf("tfer--%s_%s", tt.name, tt.id) {
				t.Fatalf("resource name = %q, want tfer--%s_%s", resource.ResourceName, tt.name, tt.id)
			}
			if resource.InstanceInfo.Type != tt.resourceType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.resourceType)
			}
		})
	}
}

func TestProductPlatformCreateResourceMissingID(t *testing.T) {
	tests := []struct {
		name   string
		create func(string) (terraformutils.Resource, error)
	}{
		{name: datadogAppBuilderAppServiceName, create: (&AppBuilderAppGenerator{}).createResource},
		{name: datadogOpenapiAPIServiceName, create: (&OpenapiAPIGenerator{}).createResource},
		{name: datadogDeploymentGateServiceName, create: (&DeploymentGateGenerator{}).createResource},
		{name: datadogDatasetServiceName, create: (&DatasetGenerator{}).createResource},
		{name: datadogDatastoreServiceName, create: (&DatastoreGenerator{}).createResource},
		{name: datadogReferenceTableServiceName, create: (&ReferenceTableGenerator{}).createResource},
		{name: datadogObservabilityPipelineServiceName, create: (&ObservabilityPipelineGenerator{}).createResource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.create(""); err == nil {
				t.Fatal("createResource returned nil error, want missing id error")
			}
		})
	}
}

func TestProductPlatformInitResourcesIDFilters(t *testing.T) {
	tests := []struct {
		name    string
		service string
		id      string
		path    string
		body    string
		factory productPlatformGeneratorFactory
	}{
		{
			name:    "app_builder_app",
			service: datadogAppBuilderAppServiceName,
			id:      "00000000-0000-0000-0000-000000000001",
			path:    "/api/v2/app-builder/apps/00000000-0000-0000-0000-000000000001",
			body:    appBuilderAppResponseJSON("00000000-0000-0000-0000-000000000001"),
			factory: newAppBuilderAppTestGenerator,
		},
		{
			name:    "openapi_api",
			service: datadogOpenapiAPIServiceName,
			id:      "00000000-0000-0000-0000-000000000010",
			path:    "/api/v2/apicatalog/api/00000000-0000-0000-0000-000000000010/openapi",
			body:    "openapi: 3.0.0\n",
			factory: newOpenapiAPITestGenerator,
		},
		{name: "deployment_gate", service: datadogDeploymentGateServiceName, id: "gate-1", path: "/api/v2/deployment_gates/gate-1", body: deploymentGateResponseJSON("gate-1"), factory: newDeploymentGateTestGenerator},
		{name: "dataset", service: datadogDatasetServiceName, id: "dataset-1", path: "/api/v2/datasets/dataset-1", body: datasetSingleResponseJSON("dataset-1"), factory: newDatasetTestGenerator},
		{name: "datastore", service: datadogDatastoreServiceName, id: "datastore-1", path: "/api/v2/actions-datastores/datastore-1", body: datastoreResponseJSON("datastore-1"), factory: newDatastoreTestGenerator},
		{name: "reference_table", service: datadogReferenceTableServiceName, id: "table-1", path: "/api/v2/reference-tables/tables/table-1", body: referenceTableResponseJSON("table-1"), factory: newReferenceTableTestGenerator},
		{name: "observability_pipeline", service: datadogObservabilityPipelineServiceName, id: "pipeline-1", path: "/api/v2/obs-pipelines/pipelines/pipeline-1", body: observabilityPipelineResponseJSON("pipeline-1"), factory: newObservabilityPipelineTestGenerator},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestedPaths := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestedPaths = append(requestedPaths, r.URL.Path)
				if r.URL.Path != tt.path {
					t.Errorf("unexpected path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				if tt.name == "openapi_api" {
					w.Header().Set("Content-Type", "application/yaml")
				} else {
					w.Header().Set("Content-Type", "application/json")
				}
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			initResources, resources := tt.factory(server, []terraformutils.ResourceFilter{{
				ServiceName:      tt.service,
				FieldPath:        "id",
				AcceptableValues: []string{tt.id},
			}})

			if err := initResources(); err != nil {
				t.Fatalf("InitResources returned error: %v", err)
			}
			if len(resources()) != 1 {
				t.Fatalf("resource count = %d, want 1", len(resources()))
			}
			if got := resources()[0].InstanceState.ID; got != tt.id {
				t.Fatalf("resource ID = %q, want %q", got, tt.id)
			}
			if len(requestedPaths) != 1 {
				t.Fatalf("requested paths = %v, want only filter read path", requestedPaths)
			}
		})
	}
}

func TestAppBuilderAppInitResourcesDoesNotTreatGlobalIDFilterAsDirectImport(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		if r.URL.Path != "/api/v2/app-builder/apps" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(appBuilderAppListResponseJSON(0)))
	}))
	t.Cleanup(server.Close)

	initResources, resources := newAppBuilderAppTestGenerator(server, []terraformutils.ResourceFilter{{
		FieldPath:        "id",
		AcceptableValues: []string{"not-a-uuid"},
	}})

	if err := initResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(resources()) != 0 {
		t.Fatalf("resource count = %d, want 0", len(resources()))
	}
	if got := strings.Join(requestedPaths, ","); got != "/api/v2/app-builder/apps" {
		t.Fatalf("requested paths = %v, want only list path", requestedPaths)
	}
}

func TestProductPlatformInitResourcesEmptyLists(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		body    string
		factory productPlatformGeneratorFactory
	}{
		{name: "app_builder_app", path: "/api/v2/app-builder/apps", body: `{"data":[],"meta":{"page":{"totalFilteredCount":0}}}`, factory: newAppBuilderAppTestGenerator},
		{name: "openapi_api", path: "/api/v2/apicatalog/api", body: `{"data":[],"meta":{"pagination":{"total_count":0}}}`, factory: newOpenapiAPITestGenerator},
		{name: "deployment_gate", path: "/api/v2/deployment_gates", body: `{"data":[],"meta":{"page":{}}}`, factory: newDeploymentGateTestGenerator},
		{name: "dataset", path: "/api/v2/datasets", body: `{"data":[]}`, factory: newDatasetTestGenerator},
		{name: "datastore", path: "/api/v2/actions-datastores", body: `{"data":[]}`, factory: newDatastoreTestGenerator},
		{name: "reference_table", path: "/api/v2/reference-tables/tables", body: `{"data":[]}`, factory: newReferenceTableTestGenerator},
		{name: "observability_pipeline", path: "/api/v2/obs-pipelines/pipelines", body: `{"data":[],"meta":{"totalCount":0}}`, factory: newObservabilityPipelineTestGenerator},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Errorf("unexpected path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			initResources, resources := tt.factory(server, nil)
			if err := initResources(); err != nil {
				t.Fatalf("InitResources returned error: %v", err)
			}
			if len(resources()) != 0 {
				t.Fatalf("resource count = %d, want 0", len(resources()))
			}
		})
	}
}

func TestProductPlatformInitResourcesPropagatesAccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	initResources, _ := newDatasetTestGenerator(server, nil)
	if err := initResources(); err == nil {
		t.Fatal("InitResources returned nil error, want access denied error")
	}
}

func TestAppBuilderAppPostConvertHookPreservesEmptyActionQueryMap(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"00000000-0000-0000-0000-000000000001",
		"app_builder_app_00000000-0000-0000-0000-000000000001",
		"datadog_app_builder_app",
		"datadog",
		AppBuilderAppAllowEmptyValues,
	)
	resource.Item = map[string]interface{}{
		"id": "00000000-0000-0000-0000-000000000001",
	}
	resource.InstanceState.Attributes = map[string]string{
		"id": "00000000-0000-0000-0000-000000000001",
		appBuilderAppActionQueryNamesToConnectionIDsKey + ".%": "0",
	}
	resource.InstanceState.SetTypedAttributes(json.RawMessage(`{"id":"00000000-0000-0000-0000-000000000001","action_query_names_to_connection_ids":{}}`))

	generator := &AppBuilderAppGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{Resources: []terraformutils.Resource{resource}},
		},
	}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	updatedResource := generator.Resources[0]
	actionMap, ok := updatedResource.Item[appBuilderAppActionQueryNamesToConnectionIDsKey].(map[string]interface{})
	if !ok {
		t.Fatalf("action query map item type = %T, want map[string]interface{}", updatedResource.Item[appBuilderAppActionQueryNamesToConnectionIDsKey])
	}
	if len(actionMap) != 0 {
		t.Fatalf("action query map length = %d, want 0", len(actionMap))
	}
	if got := updatedResource.InstanceState.Attributes[appBuilderAppActionQueryNamesToConnectionIDsKey+".%"]; got != "0" {
		t.Fatalf("state action query map count = %q, want 0", got)
	}
	typedAttributes := decodeProductPlatformTypedAttributes(t, updatedResource.InstanceState.TypedAttributes)
	if got := string(typedAttributes[appBuilderAppActionQueryNamesToConnectionIDsKey]); got != "{}" {
		t.Fatalf("typed action query map = %s, want {}", got)
	}
}

func TestAppBuilderAppInitResourcesListsPages(t *testing.T) {
	pages := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/app-builder/apps" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("limit"); got != fmt.Sprint(datadogAppBuilderAppPageLimit) {
			t.Errorf("limit query = %q, want %d", got, datadogAppBuilderAppPageLimit)
		}
		page := r.URL.Query().Get("page")
		pages = append(pages, page)
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "0":
			_, _ = w.Write([]byte(appBuilderAppListResponseJSON(2, "00000000-0000-0000-0000-000000000001")))
		case "1":
			_, _ = w.Write([]byte(appBuilderAppListResponseJSON(2, "00000000-0000-0000-0000-000000000002")))
		default:
			t.Errorf("unexpected page %q", page)
			_, _ = w.Write([]byte(appBuilderAppListResponseJSON(2)))
		}
	}))
	t.Cleanup(server.Close)

	initResources, resources := newAppBuilderAppTestGenerator(server, nil)
	if err := initResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertProductPlatformResourceIDs(t, resources(), []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"})
	if strings.Join(pages, ",") != "0,1" {
		t.Fatalf("pages = %v, want [0 1]", pages)
	}
}

func TestOpenapiAPIInitResourcesListsPages(t *testing.T) {
	offsets := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/apicatalog/api" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("page[limit]"); got != fmt.Sprint(datadogOpenapiAPIPageLimit) {
			t.Errorf("page[limit] query = %q, want %d", got, datadogOpenapiAPIPageLimit)
		}
		offset := r.URL.Query().Get("page[offset]")
		offsets = append(offsets, offset)
		w.Header().Set("Content-Type", "application/json")
		switch offset {
		case "0":
			_, _ = w.Write([]byte(openapiAPIListResponseJSON(2, "00000000-0000-0000-0000-000000000010")))
		case "100":
			_, _ = w.Write([]byte(openapiAPIListResponseJSON(2, "00000000-0000-0000-0000-000000000011")))
		default:
			t.Errorf("unexpected offset %q", offset)
			_, _ = w.Write([]byte(openapiAPIListResponseJSON(2)))
		}
	}))
	t.Cleanup(server.Close)

	initResources, resources := newOpenapiAPITestGenerator(server, nil)
	if err := initResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertProductPlatformResourceIDs(t, resources(), []string{"00000000-0000-0000-0000-000000000010", "00000000-0000-0000-0000-000000000011"})
	if strings.Join(offsets, ",") != "0,100" {
		t.Fatalf("offsets = %v, want [0 100]", offsets)
	}
}

func TestDeploymentGateInitResourcesListsCursorPages(t *testing.T) {
	cursors := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/deployment_gates" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("page[size]"); got != fmt.Sprint(datadogDeploymentGatePageSize) {
			t.Errorf("page[size] query = %q, want %d", got, datadogDeploymentGatePageSize)
		}
		cursor := r.URL.Query().Get("page[cursor]")
		cursors = append(cursors, cursor)
		w.Header().Set("Content-Type", "application/json")
		switch cursor {
		case "":
			_, _ = w.Write([]byte(deploymentGateListResponseJSON("cursor-2", "gate-1")))
		case "cursor-2":
			_, _ = w.Write([]byte(deploymentGateListResponseJSON("", "gate-2")))
		default:
			t.Errorf("unexpected cursor %q", cursor)
			_, _ = w.Write([]byte(deploymentGateListResponseJSON("")))
		}
	}))
	t.Cleanup(server.Close)

	initResources, resources := newDeploymentGateTestGenerator(server, nil)
	if err := initResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertProductPlatformResourceIDs(t, resources(), []string{"gate-1", "gate-2"})
	if strings.Join(cursors, ",") != ",cursor-2" {
		t.Fatalf("cursors = %v, want [ cursor-2]", cursors)
	}
}

func TestDatasetInitResourcesList(t *testing.T) {
	server := productPlatformListServer(t, "/api/v2/datasets", datasetListResponseJSON("dataset-1", "dataset-2"))
	initResources, resources := newDatasetTestGenerator(server, nil)
	if err := initResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertProductPlatformResourceIDs(t, resources(), []string{"dataset-1", "dataset-2"})
}

func TestDatastoreInitResourcesList(t *testing.T) {
	server := productPlatformListServer(t, "/api/v2/actions-datastores", datastoreListResponseJSON("datastore-1", "datastore-2"))
	initResources, resources := newDatastoreTestGenerator(server, nil)
	if err := initResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertProductPlatformResourceIDs(t, resources(), []string{"datastore-1", "datastore-2"})
}

func TestReferenceTableInitResourcesListsPages(t *testing.T) {
	offsets := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/reference-tables/tables" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("page[limit]"); got != fmt.Sprint(datadogReferenceTablePageLimit) {
			t.Errorf("page[limit] query = %q, want %d", got, datadogReferenceTablePageLimit)
		}
		offset := r.URL.Query().Get("page[offset]")
		offsets = append(offsets, offset)
		w.Header().Set("Content-Type", "application/json")
		switch offset {
		case "0":
			_, _ = w.Write([]byte(referenceTableListResponseJSON(repeatedIDs("table", int(datadogReferenceTablePageLimit))...)))
		case "100":
			_, _ = w.Write([]byte(referenceTableListResponseJSON("table-100")))
		default:
			t.Errorf("unexpected offset %q", offset)
			_, _ = w.Write([]byte(referenceTableListResponseJSON()))
		}
	}))
	t.Cleanup(server.Close)

	initResources, resources := newReferenceTableTestGenerator(server, nil)
	if err := initResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(resources()) != int(datadogReferenceTablePageLimit)+1 {
		t.Fatalf("resource count = %d, want %d", len(resources()), int(datadogReferenceTablePageLimit)+1)
	}
	if strings.Join(offsets, ",") != "0,100" {
		t.Fatalf("offsets = %v, want [0 100]", offsets)
	}
}

func TestReferenceTableInitResourcesSkipsUnsupportedLocalFileSource(t *testing.T) {
	server := productPlatformListServer(
		t,
		"/api/v2/reference-tables/tables",
		fmt.Sprintf(`{"data":[%s,%s]}`, referenceTableDataJSONWithSource("table-s3", "S3"), referenceTableDataJSONWithSource("table-local", "LOCAL_FILE")),
	)

	initResources, resources := newReferenceTableTestGenerator(server, nil)
	if err := initResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertProductPlatformResourceIDs(t, resources(), []string{"table-s3"})
}

func TestObservabilityPipelineInitResourcesListsPages(t *testing.T) {
	pages := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/obs-pipelines/pipelines" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("page[size]"); got != fmt.Sprint(datadogObservabilityPipelinePageSize) {
			t.Errorf("page[size] query = %q, want %d", got, datadogObservabilityPipelinePageSize)
		}
		page := r.URL.Query().Get("page[number]")
		pages = append(pages, page)
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "0":
			_, _ = w.Write([]byte(observabilityPipelineListResponseJSON(2, "pipeline-1")))
		case "1":
			_, _ = w.Write([]byte(observabilityPipelineListResponseJSON(2, "pipeline-2")))
		default:
			t.Errorf("unexpected page %q", page)
			_, _ = w.Write([]byte(observabilityPipelineListResponseJSON(2)))
		}
	}))
	t.Cleanup(server.Close)

	initResources, resources := newObservabilityPipelineTestGenerator(server, nil)
	if err := initResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertProductPlatformResourceIDs(t, resources(), []string{"pipeline-1", "pipeline-2"})
	if strings.Join(pages, ",") != "0,1" {
		t.Fatalf("pages = %v, want [0 1]", pages)
	}
}

func newProductPlatformDatadogService(server *httptest.Server, filter []terraformutils.ResourceFilter) DatadogService {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return DatadogService{
		Service: terraformutils.Service{
			Args: map[string]interface{}{
				"auth":          context.Background(),
				"datadogClient": datadog.NewAPIClient(config),
			},
			Filter: filter,
		},
	}
}

func newAppBuilderAppTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) (func() error, func() []terraformutils.Resource) {
	generator := &AppBuilderAppGenerator{DatadogService: newProductPlatformDatadogService(server, filter)}
	return generator.InitResources, func() []terraformutils.Resource { return generator.Resources }
}

func newOpenapiAPITestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) (func() error, func() []terraformutils.Resource) {
	generator := &OpenapiAPIGenerator{DatadogService: newProductPlatformDatadogService(server, filter)}
	return generator.InitResources, func() []terraformutils.Resource { return generator.Resources }
}

func newDeploymentGateTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) (func() error, func() []terraformutils.Resource) {
	generator := &DeploymentGateGenerator{DatadogService: newProductPlatformDatadogService(server, filter)}
	return generator.InitResources, func() []terraformutils.Resource { return generator.Resources }
}

func newDatasetTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) (func() error, func() []terraformutils.Resource) {
	generator := &DatasetGenerator{DatadogService: newProductPlatformDatadogService(server, filter)}
	return generator.InitResources, func() []terraformutils.Resource { return generator.Resources }
}

func newDatastoreTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) (func() error, func() []terraformutils.Resource) {
	generator := &DatastoreGenerator{DatadogService: newProductPlatformDatadogService(server, filter)}
	return generator.InitResources, func() []terraformutils.Resource { return generator.Resources }
}

func newReferenceTableTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) (func() error, func() []terraformutils.Resource) {
	generator := &ReferenceTableGenerator{DatadogService: newProductPlatformDatadogService(server, filter)}
	return generator.InitResources, func() []terraformutils.Resource { return generator.Resources }
}

func newObservabilityPipelineTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) (func() error, func() []terraformutils.Resource) {
	generator := &ObservabilityPipelineGenerator{DatadogService: newProductPlatformDatadogService(server, filter)}
	return generator.InitResources, func() []terraformutils.Resource { return generator.Resources }
}

func productPlatformListServer(t *testing.T, path, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func assertProductPlatformResourceIDs(t *testing.T, resources []terraformutils.Resource, want []string) {
	t.Helper()

	if len(resources) != len(want) {
		t.Fatalf("resource count = %d, want %d", len(resources), len(want))
	}
	for i, resource := range resources {
		if resource.InstanceState.ID != want[i] {
			t.Fatalf("resource[%d] ID = %q, want %q", i, resource.InstanceState.ID, want[i])
		}
	}
}

func decodeProductPlatformTypedAttributes(t *testing.T, rawAttributes json.RawMessage) map[string]json.RawMessage {
	t.Helper()

	attributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(rawAttributes, &attributes); err != nil {
		t.Fatalf("typed attributes unmarshal error: %v", err)
	}
	return attributes
}

func appBuilderAppResponseJSON(id string) string {
	return fmt.Sprintf(`{"data":{"id":%q,"type":"appDefinitions","attributes":{}}}`, id)
}

func appBuilderAppListResponseJSON(total int, ids ...string) string {
	apps := make([]string, 0, len(ids))
	for _, id := range ids {
		apps = append(apps, fmt.Sprintf(`{"id":%q,"type":"appDefinitions","attributes":{}}`, id))
	}
	return fmt.Sprintf(`{"data":[%s],"meta":{"page":{"totalFilteredCount":%d}}}`, strings.Join(apps, ","), total)
}

func openapiAPIListResponseJSON(total int, ids ...string) string {
	apis := make([]string, 0, len(ids))
	for _, id := range ids {
		apis = append(apis, fmt.Sprintf(`{"id":%q}`, id))
	}
	return fmt.Sprintf(`{"data":[%s],"meta":{"pagination":{"total_count":%d}}}`, strings.Join(apis, ","), total)
}

func deploymentGateResponseJSON(id string) string {
	return fmt.Sprintf(`{"data":%s}`, deploymentGateDataJSON(id))
}

func deploymentGateListResponseJSON(nextCursor string, ids ...string) string {
	gates := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		gates = append(gates, deploymentGateDataJSON(id))
	}
	page := `{}`
	if nextCursor != "" {
		page = fmt.Sprintf(`{"next_cursor":%q}`, nextCursor)
	}
	return fmt.Sprintf(`{"data":[%s],"meta":{"page":%s}}`, strings.Join(gates, ","), page)
}

func deploymentGateDataJSON(id string) string {
	return fmt.Sprintf(`{"id":%q,"type":"deployment_gate","attributes":{"created_at":"2024-01-01T00:00:00Z","created_by":{"id":"user-1"},"dry_run":false,"env":"prod","identifier":"gate","service":"svc"}}`, id)
}

func datasetSingleResponseJSON(id string) string {
	return fmt.Sprintf(`{"data":{"id":%q,"type":"dataset","attributes":{"name":"dataset","principals":["role:abc"]}}}`, id)
}

func datasetListResponseJSON(ids ...string) string {
	datasets := make([]string, 0, len(ids))
	for _, id := range ids {
		datasets = append(datasets, fmt.Sprintf(`{"id":%q,"type":"dataset","attributes":{"name":"dataset","principals":["role:abc"]}}`, id))
	}
	return fmt.Sprintf(`{"data":[%s]}`, strings.Join(datasets, ","))
}

func datastoreResponseJSON(id string) string {
	return fmt.Sprintf(`{"data":%s}`, datastoreDataJSON(id))
}

func datastoreListResponseJSON(ids ...string) string {
	datastores := make([]string, 0, len(ids))
	for _, id := range ids {
		datastores = append(datastores, datastoreDataJSON(id))
	}
	return fmt.Sprintf(`{"data":[%s]}`, strings.Join(datastores, ","))
}

func datastoreDataJSON(id string) string {
	return fmt.Sprintf(`{"id":%q,"type":"datastores","attributes":{"name":"datastore","primary_column_name":"id"}}`, id)
}

func referenceTableResponseJSON(id string) string {
	return fmt.Sprintf(`{"data":%s}`, referenceTableDataJSON(id))
}

func referenceTableListResponseJSON(ids ...string) string {
	tables := make([]string, 0, len(ids))
	for _, id := range ids {
		tables = append(tables, referenceTableDataJSON(id))
	}
	return fmt.Sprintf(`{"data":[%s]}`, strings.Join(tables, ","))
}

func referenceTableDataJSON(id string) string {
	return referenceTableDataJSONWithSource(id, "S3")
}

func referenceTableDataJSONWithSource(id, source string) string {
	return fmt.Sprintf(`{"id":%q,"type":"reference_table","attributes":{"table_name":"table","source":%q}}`, id, source)
}

func observabilityPipelineResponseJSON(id string) string {
	return fmt.Sprintf(`{"data":%s}`, observabilityPipelineDataJSON(id))
}

func observabilityPipelineListResponseJSON(total int, ids ...string) string {
	pipelines := make([]string, 0, len(ids))
	for _, id := range ids {
		pipelines = append(pipelines, observabilityPipelineDataJSON(id))
	}
	return fmt.Sprintf(`{"data":[%s],"meta":{"totalCount":%d}}`, strings.Join(pipelines, ","), total)
}

func observabilityPipelineDataJSON(id string) string {
	return fmt.Sprintf(`{"id":%q,"type":"pipelines","attributes":{"name":"pipeline","config":{"sources":[],"destinations":[]}}}`, id)
}

func repeatedIDs(prefix string, count int) []string {
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		ids = append(ids, fmt.Sprintf("%s-%d", prefix, i))
	}
	return ids
}
