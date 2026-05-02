// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestTeamConnectionCreateResource(t *testing.T) {
	generator := &TeamConnectionGenerator{}
	resource, err := generator.createResource(teamConnection("connection-1", "team-1", "github-team-1"))
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "connection-1" {
		t.Fatalf("resource ID = %q, want connection-1", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--team_connection_connection-1" {
		t.Fatalf("resource name = %q, want tfer--team_connection_connection-1", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_team_connection" {
		t.Fatalf("resource type = %q, want datadog_team_connection", resource.InstanceInfo.Type)
	}
	if resource.InstanceState.Attributes["team.id"] != "team-1" {
		t.Fatalf("team.id = %q, want team-1", resource.InstanceState.Attributes["team.id"])
	}
	if resource.InstanceState.Attributes["connected_team.id"] != "github-team-1" {
		t.Fatalf("connected_team.id = %q, want github-team-1", resource.InstanceState.Attributes["connected_team.id"])
	}
}

func TestTeamConnectionCreateResourceRequiresID(t *testing.T) {
	generator := &TeamConnectionGenerator{}
	if _, err := generator.createResource(teamConnection("", "team-1", "github-team-1")); err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
	if _, err := generator.createResource(teamConnection("connection-1", "", "github-team-1")); err == nil {
		t.Fatal("createResource returned nil error, want missing team id error")
	}
	if _, err := generator.createResource(teamConnection("connection-1", "team-1", "")); err == nil {
		t.Fatal("createResource returned nil error, want missing connected team id error")
	}
}

func TestTeamConnectionInitResourcesListsConnections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/team/connections" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, teamConnectionListResponseJSON(
			teamConnectionJSON("connection-1", "team-1", "github-team-1"),
			teamConnectionJSON("connection-2", "team-2", "github-team-2"),
		))
	}))
	defer server.Close()

	generator := newTeamConnectionTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "connection-1" || generator.Resources[1].InstanceState.ID != "connection-2" {
		t.Fatalf("unexpected resource IDs: %s, %s", generator.Resources[0].InstanceState.ID, generator.Resources[1].InstanceState.ID)
	}
}

func TestTeamConnectionInitResourcesFiltersByID(t *testing.T) {
	filterCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/team/connections" {
			http.NotFound(w, r)
			return
		}
		filterCh <- r.URL.Query().Get("filter[connection_ids]")
		_, _ = fmt.Fprint(w, teamConnectionListResponseJSON(teamConnectionJSON("connection-1", "team-1", "github-team-1")))
	}))
	defer server.Close()

	generator := newTeamConnectionTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "team_connection",
			FieldPath:        "id",
			AcceptableValues: []string{"connection-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, filterCh, "filter[connection_ids]", "connection-1")
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "connection-1" {
		t.Fatalf("resource ID = %q, want connection-1", generator.Resources[0].InstanceState.ID)
	}
}

func TestTeamConnectionInitResourcesFiltersBySource(t *testing.T) {
	filterCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/team/connections" {
			http.NotFound(w, r)
			return
		}
		filterCh <- r.URL.Query().Get("filter[sources]")
		_, _ = fmt.Fprint(w, teamConnectionListResponseJSON(teamConnectionJSON("connection-1", "team-1", "github-team-1")))
	}))
	defer server.Close()

	generator := newTeamConnectionTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "team_connection",
			FieldPath:        "source",
			AcceptableValues: []string{"github"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, filterCh, "filter[sources]", "github")
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
}

func assertObservedQueryValue(t *testing.T, observed <-chan string, name string, want string) {
	t.Helper()

	select {
	case got := <-observed:
		if got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s query value", name)
	}
}

func newTeamConnectionTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *TeamConnectionGenerator {
	return &TeamConnectionGenerator{
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

func newTeamRelationshipTestClient(server *httptest.Server) *datadog.APIClient {
	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()
	return datadog.NewAPIClient(config)
}

func teamConnectionListResponseJSON(connections ...string) string {
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(connections, ","))
}

func teamConnection(id string, teamID string, connectedTeamID string) datadogV2.TeamConnection {
	teamRef := datadogV2.NewTeamRef()
	teamRef.SetData(*datadogV2.NewTeamRefData(teamID, datadogV2.TEAMREFDATATYPE_TEAM))
	connectedTeamRef := datadogV2.NewConnectedTeamRef()
	connectedTeamRef.SetData(*datadogV2.NewConnectedTeamRefData(connectedTeamID, datadogV2.CONNECTEDTEAMREFDATATYPE_GITHUB_TEAM))
	relationships := datadogV2.NewTeamConnectionRelationships()
	relationships.SetTeam(*teamRef)
	relationships.SetConnectedTeam(*connectedTeamRef)
	attributes := datadogV2.NewTeamConnectionAttributes()
	attributes.SetSource("github")
	teamConnection := datadogV2.NewTeamConnection(id, datadogV2.TEAMCONNECTIONTYPE_TEAM_CONNECTION)
	teamConnection.SetRelationships(*relationships)
	teamConnection.SetAttributes(*attributes)
	return *teamConnection
}

func teamConnectionJSON(id string, teamID string, connectedTeamID string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"team_connection\",\"attributes\":{\"source\":%q},\"relationships\":{\"team\":{\"data\":{\"id\":%q,\"type\":\"team\"}},\"connected_team\":{\"data\":{\"id\":%q,\"type\":\"github_team\"}}}}",
		id,
		"github",
		teamID,
		connectedTeamID,
	)
}
