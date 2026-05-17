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

func TestServiceDefinitionYAMLCreateResource(t *testing.T) {
	resource, err := (&ServiceDefinitionYAMLGenerator{}).createResource("checkout")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}
	if resource.InstanceState.ID != "checkout" {
		t.Fatalf("resource ID = %q, want checkout", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--service_definition_yaml_checkout" {
		t.Fatalf("resource name = %q, want tfer--service_definition_yaml_checkout", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_service_definition_yaml" {
		t.Fatalf("resource type = %q, want datadog_service_definition_yaml", resource.InstanceInfo.Type)
	}
}

func TestServiceDefinitionYAMLCreateResourceMissingID(t *testing.T) {
	if _, err := (&ServiceDefinitionYAMLGenerator{}).createResource(""); err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestServiceDefinitionYAMLInitResourcesListsDefinitionsWithPagination(t *testing.T) {
	requestPages := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/services/definitions" {
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
			_, _ = fmt.Fprint(w, serviceDefinitionListResponseJSON(serviceNames("service", 0, 100)...))
		case "1":
			_, _ = fmt.Fprint(w, serviceDefinitionListResponseJSON("service-100"))
		default:
			http.Error(w, fmt.Sprintf("page[number] = %q, want 0 or 1", pageNumber), http.StatusBadRequest)
		}
	}))
	defer server.Close()

	generator := newServiceDefinitionYAMLTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if strings.Join(requestPages, ",") != "0,1" {
		t.Fatalf("request pages = %v, want [0 1]", requestPages)
	}
	assertResourceIDs(t, generator.Resources, serviceNames("service", 0, 101))
}

func TestServiceDefinitionYAMLInitResourcesFiltersByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/services/definitions/checkout" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, serviceDefinitionGetResponseJSON("checkout"))
	}))
	defer server.Close()

	generator := newServiceDefinitionYAMLTestGenerator(server, []terraformutils.ResourceFilter{{
		ServiceName:      "service_definition_yaml",
		FieldPath:        "id",
		AcceptableValues: []string{"checkout"},
	}})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"checkout"})
}

func TestServiceDefinitionYAMLInitResourcesListsWithUnrelatedIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/services/definitions" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, serviceDefinitionListResponseJSON("checkout"))
	}))
	defer server.Close()

	generator := newServiceDefinitionYAMLTestGenerator(server, []terraformutils.ResourceFilter{{
		ServiceName:      "team",
		FieldPath:        "id",
		AcceptableValues: []string{"team-id"},
	}})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"checkout"})
}

func TestSoftwareCatalogCreateResource(t *testing.T) {
	resource, err := (&SoftwareCatalogGenerator{}).createResource("service:default/checkout")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}
	if resource.InstanceState.ID != "service:default/checkout" {
		t.Fatalf("resource ID = %q, want service:default/checkout", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--software_catalog_service-003A-default-002F-checkout" {
		t.Fatalf("resource name = %q, want tfer--software_catalog_service-003A-default-002F-checkout", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_software_catalog" {
		t.Fatalf("resource type = %q, want datadog_software_catalog", resource.InstanceInfo.Type)
	}
}

func TestSoftwareCatalogCreateResourceMissingID(t *testing.T) {
	if _, err := (&SoftwareCatalogGenerator{}).createResource(""); err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestSoftwareCatalogInitResourcesListsImportableEntitiesWithPagination(t *testing.T) {
	requestOffsets := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/catalog/entity" {
			http.NotFound(w, r)
			return
		}
		if errMessage := validateSoftwareCatalogListQuery(r); errMessage != "" {
			http.Error(w, errMessage, http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("page[limit]"); got != "100" {
			http.Error(w, fmt.Sprintf("page[limit] = %q, want 100", got), http.StatusBadRequest)
			return
		}
		offset := r.URL.Query().Get("page[offset]")
		requestOffsets = append(requestOffsets, offset)
		switch offset {
		case "0":
			_, _ = fmt.Fprint(w, softwareCatalogListResponseJSON(softwareCatalogRefs("service", 0, 100)...))
		case "100":
			_, _ = fmt.Fprint(w, softwareCatalogListResponseJSON("service:default/service-100"))
		default:
			http.Error(w, fmt.Sprintf("page[offset] = %q, want 0 or 100", offset), http.StatusBadRequest)
		}
	}))
	defer server.Close()

	generator := newSoftwareCatalogTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if strings.Join(requestOffsets, ",") != "0,100" {
		t.Fatalf("request offsets = %v, want [0 100]", requestOffsets)
	}
	assertResourceIDs(t, generator.Resources, softwareCatalogRefs("service", 0, 101))
}

func TestSoftwareCatalogInitResourcesFiltersByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/catalog/entity" {
			http.NotFound(w, r)
			return
		}
		if errMessage := validateSoftwareCatalogListQuery(r); errMessage != "" {
			http.Error(w, errMessage, http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("filter[ref]"); got != "service:default/checkout" {
			http.Error(w, fmt.Sprintf("filter[ref] = %q, want service:default/checkout", got), http.StatusBadRequest)
			return
		}
		_, _ = fmt.Fprint(w, softwareCatalogListResponseJSON("service:default/checkout"))
	}))
	defer server.Close()

	generator := newSoftwareCatalogTestGenerator(server, []terraformutils.ResourceFilter{{
		ServiceName:      "software_catalog",
		FieldPath:        "id",
		AcceptableValues: []string{"service:default/checkout"},
	}})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"service:default/checkout"})
}

func TestSoftwareCatalogInitResourcesSkipsEntitiesWithoutRawSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/catalog/entity" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, softwareCatalogMixedRawSchemaResponseJSON())
	}))
	defer server.Close()

	generator := newSoftwareCatalogTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"service:default/with-raw"})
}

func TestSoftwareCatalogInitResourcesErrorsForFilteredEntityWithoutRawSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/catalog/entity" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, softwareCatalogRawlessResponseJSON("service:default/without-raw"))
	}))
	defer server.Close()

	generator := newSoftwareCatalogTestGenerator(server, []terraformutils.ResourceFilter{{
		ServiceName:      "software_catalog",
		FieldPath:        "id",
		AcceptableValues: []string{"service:default/without-raw"},
	}})
	if err := generator.InitResources(); err == nil {
		t.Fatal("InitResources returned nil error, want raw schema error")
	}
}

func TestSoftwareCatalogEntityRefDefaultsNamespace(t *testing.T) {
	ref, err := softwareCatalogEntityRef(softwareCatalogEntity("service:default/checkout", "raw-checkout", true, false))
	if err != nil {
		t.Fatalf("softwareCatalogEntityRef returned error: %v", err)
	}
	if ref != "service:default/checkout" {
		t.Fatalf("ref = %q, want service:default/checkout", ref)
	}
}

func validateSoftwareCatalogListQuery(r *http.Request) string {
	if got := r.URL.Query().Get("include"); got != "raw_schema" {
		return fmt.Sprintf("include = %q, want raw_schema", got)
	}
	if got := r.URL.Query().Get("includeDiscovered"); got != "false" {
		return fmt.Sprintf("includeDiscovered = %q, want false", got)
	}
	if got := r.URL.Query().Get("filter[exclude_snapshot]"); got != "true" {
		return fmt.Sprintf("filter[exclude_snapshot] = %q, want true", got)
	}
	return ""
}

func newServiceDefinitionYAMLTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *ServiceDefinitionYAMLGenerator {
	return &ServiceDefinitionYAMLGenerator{DatadogService: newServiceCatalogTestService(server, filter)}
}

func newSoftwareCatalogTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *SoftwareCatalogGenerator {
	return &SoftwareCatalogGenerator{DatadogService: newServiceCatalogTestService(server, filter)}
}

func newServiceCatalogTestService(server *httptest.Server, filter []terraformutils.ResourceFilter) DatadogService {
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

func serviceNames(prefix string, start, count int) []string {
	names := make([]string, 0, count)
	for i := start; i < start+count; i++ {
		names = append(names, fmt.Sprintf("%s-%d", prefix, i))
	}
	return names
}

func serviceDefinitionGetResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":%s}", serviceDefinitionJSON(id))
}

func serviceDefinitionListResponseJSON(ids ...string) string {
	items := make([]string, 0, len(ids))
	for _, id := range ids {
		items = append(items, serviceDefinitionJSON(id))
	}
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(items, ","))
}

func serviceDefinitionJSON(id string) string {
	return fmt.Sprintf("{\"id\":%q,\"type\":\"service_definitions\"}", id)
}

func softwareCatalogRefs(kind string, start, count int) []string {
	refs := make([]string, 0, count)
	for i := start; i < start+count; i++ {
		refs = append(refs, fmt.Sprintf("%s:default/%s-%d", kind, kind, i))
	}
	return refs
}

func softwareCatalogListResponseJSON(refs ...string) string {
	data := make([]string, 0, len(refs))
	included := make([]string, 0, len(refs))
	for _, ref := range refs {
		rawSchemaID := strings.NewReplacer(":", "-", "/", "-").Replace(ref)
		data = append(data, softwareCatalogEntityJSON(ref, rawSchemaID, true, true))
		included = append(included, softwareCatalogRawSchemaJSON(rawSchemaID))
	}
	return fmt.Sprintf("{\"data\":[%s],\"included\":[%s]}", strings.Join(data, ","), strings.Join(included, ","))
}

func softwareCatalogMixedRawSchemaResponseJSON() string {
	return fmt.Sprintf(
		"{\"data\":[%s,%s],\"included\":[%s]}",
		softwareCatalogEntityJSON("service:default/with-raw", "raw-with-raw", true, true),
		softwareCatalogEntityJSON("service:default/without-raw", "raw-without-raw", true, false),
		softwareCatalogRawSchemaJSON("raw-with-raw"),
	)
}

func softwareCatalogRawlessResponseJSON(ref string) string {
	return fmt.Sprintf("{\"data\":[%s],\"included\":[]}", softwareCatalogEntityJSON(ref, "", false, true))
}

func softwareCatalogEntityJSON(ref, rawSchemaID string, includeRelationship bool, includeNamespace bool) string {
	kind, namespace, name := splitCatalogRef(ref)
	namespaceAttr := ""
	if includeNamespace {
		namespaceAttr = fmt.Sprintf(",\"namespace\":%q", namespace)
	}
	relationships := ""
	if includeRelationship {
		relationships = fmt.Sprintf(",\"relationships\":{\"rawSchema\":{\"data\":{\"id\":%q,\"type\":\"rawSchema\"}}}", rawSchemaID)
	}
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"entities\",\"attributes\":{\"kind\":%q,\"name\":%q%s}%s}",
		fmt.Sprintf("entity-%s", rawSchemaID),
		kind,
		name,
		namespaceAttr,
		relationships,
	)
}

func softwareCatalogRawSchemaJSON(id string) string {
	return fmt.Sprintf("{\"id\":%q,\"type\":\"rawSchema\",\"attributes\":{\"rawSchema\":\"YXBpVmVyc2lvbjogdjM=\"}}", id)
}

func softwareCatalogEntity(ref string, rawSchemaID string, includeRelationship bool, includeNamespace bool) datadogV2.EntityData {
	kind, namespace, name := splitCatalogRef(ref)
	attributes := datadogV2.NewEntityAttributes()
	attributes.SetKind(kind)
	attributes.SetName(name)
	if includeNamespace {
		attributes.SetNamespace(namespace)
	}
	entity := datadogV2.NewEntityData()
	entity.SetAttributes(*attributes)
	if includeRelationship {
		relationshipItem := datadogV2.NewRelationshipItem()
		relationshipItem.SetId(rawSchemaID)
		rawSchema := datadogV2.NewEntityToRawSchema()
		rawSchema.SetData(*relationshipItem)
		relationships := datadogV2.NewEntityRelationships()
		relationships.SetRawSchema(*rawSchema)
		entity.SetRelationships(*relationships)
	}
	return *entity
}

func splitCatalogRef(ref string) (string, string, string) {
	kind, rest, _ := strings.Cut(ref, ":")
	namespace, name, _ := strings.Cut(rest, "/")
	return kind, namespace, name
}
