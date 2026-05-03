// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestTeamSyncCreateResource(t *testing.T) {
	attributes := datadogV2.NewTeamSyncAttributes(datadogV2.TEAMSYNCATTRIBUTESSOURCE_GITHUB, datadogV2.TEAMSYNCATTRIBUTESTYPE_LINK)
	teamSync := datadogV2.NewTeamSyncData(*attributes, datadogV2.TEAMSYNCBULKTYPE_TEAM_SYNC_BULK)

	generator := &TeamSyncGenerator{}
	resource, err := generator.createResource(*teamSync)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}
	if resource.InstanceState.ID != "github" {
		t.Fatalf("resource ID = %q, want github", resource.InstanceState.ID)
	}
	if resource.InstanceState.Attributes["source"] != "github" {
		t.Fatalf("source = %q, want github", resource.InstanceState.Attributes["source"])
	}
	if resource.InstanceState.Attributes["type"] != "link" {
		t.Fatalf("type = %q, want link", resource.InstanceState.Attributes["type"])
	}
	if resource.ResourceName != "tfer--team_sync_github" {
		t.Fatalf("resource name = %q, want tfer--team_sync_github", resource.ResourceName)
	}
}

func TestTeamSyncCreateResourceRequiresSourceAndType(t *testing.T) {
	generator := &TeamSyncGenerator{}
	if _, err := generator.createResource(datadogV2.TeamSyncData{}); err == nil {
		t.Fatal("createResource returned nil error, want missing source error")
	}

	teamSync := datadogV2.TeamSyncData{
		Attributes: datadogV2.TeamSyncAttributes{
			Source: datadogV2.TEAMSYNCATTRIBUTESSOURCE_GITHUB,
		},
	}
	if _, err := generator.createResource(teamSync); err == nil {
		t.Fatal("createResource returned nil error, want missing type error")
	}
}

func TestTeamSyncInitResourcesGetsGitHubSource(t *testing.T) {
	filterCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/team/sync" {
			http.NotFound(w, r)
			return
		}
		filterCh <- r.URL.Query().Get("filter[source]")
		_, _ = fmt.Fprint(w, teamSyncResponseJSON(teamSyncJSON("github", "link")))
	}))
	defer server.Close()

	generator := newTeamSyncTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, filterCh, "filter[source]", "github")
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "github" {
		t.Fatalf("resource ID = %q, want github", generator.Resources[0].InstanceState.ID)
	}
}

func TestTeamSyncInitResourcesSkipsEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, "{\"data\":[]}")
	}))
	defer server.Close()

	generator := newTeamSyncTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(generator.Resources))
	}
}

func TestTeamSyncInitResourcesSkipsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	generator := newTeamSyncTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(generator.Resources))
	}
}

func TestTeamSyncInitResourcesFiltersBySource(t *testing.T) {
	filterCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		filterCh <- r.URL.Query().Get("filter[source]")
		_, _ = fmt.Fprint(w, teamSyncResponseJSON(teamSyncJSON("github", "provision")))
	}))
	defer server.Close()

	generator := newTeamSyncTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "team_sync",
			FieldPath:        "source",
			AcceptableValues: []string{"github"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, filterCh, "filter[source]", "github")
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
}

func newTeamSyncTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *TeamSyncGenerator {
	return &TeamSyncGenerator{
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

func teamSyncResponseJSON(teamSync string) string {
	return fmt.Sprintf("{\"data\":[%s]}", teamSync)
}

func teamSyncJSON(source string, syncType string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"team_sync_bulk\",\"attributes\":{\"source\":%q,\"type\":%q,\"frequency\":\"once\",\"sync_membership\":false}}",
		source,
		source,
		syncType,
	)
}
