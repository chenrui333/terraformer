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

func TestTeamHierarchyLinksCreateResource(t *testing.T) {
	generator := &TeamHierarchyLinksGenerator{}
	resource, err := generator.createResource(teamHierarchyLink("link-1", "parent-team", "sub-team"))
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "link-1" {
		t.Fatalf("resource ID = %q, want link-1", resource.InstanceState.ID)
	}
	if resource.InstanceState.Attributes["parent_team_id"] != "parent-team" {
		t.Fatalf("parent_team_id = %q, want parent-team", resource.InstanceState.Attributes["parent_team_id"])
	}
	if resource.InstanceState.Attributes["sub_team_id"] != "sub-team" {
		t.Fatalf("sub_team_id = %q, want sub-team", resource.InstanceState.Attributes["sub_team_id"])
	}
	if resource.ResourceName != "tfer--team_hierarchy_links_parent-team_sub-team" {
		t.Fatalf("resource name = %q, want tfer--team_hierarchy_links_parent-team_sub-team", resource.ResourceName)
	}
}

func TestTeamHierarchyLinksCreateResourceRequiresRelationships(t *testing.T) {
	generator := &TeamHierarchyLinksGenerator{}
	if _, err := generator.createResource(teamHierarchyLink("", "parent-team", "sub-team")); err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
	if _, err := generator.createResource(teamHierarchyLink("link-1", "", "sub-team")); err == nil {
		t.Fatal("createResource returned nil error, want missing parent team id error")
	}
	if _, err := generator.createResource(teamHierarchyLink("link-1", "parent-team", "")); err == nil {
		t.Fatal("createResource returned nil error, want missing sub team id error")
	}
}

func TestTeamHierarchyLinksInitResourcesListsLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/team-hierarchy-links" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, teamHierarchyLinksListResponseJSON(
			teamHierarchyLinkJSON("link-1", "parent-1", "sub-1"),
			teamHierarchyLinkJSON("link-2", "parent-2", "sub-2"),
		))
	}))
	defer server.Close()

	generator := newTeamHierarchyLinksTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(generator.Resources))
	}
}

func TestTeamHierarchyLinksInitResourcesFiltersByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/team-hierarchy-links/link-1" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintf(w, "{\"data\":%s}", teamHierarchyLinkJSON("link-1", "parent-1", "sub-1"))
	}))
	defer server.Close()

	generator := newTeamHierarchyLinksTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "team_hierarchy_links",
			FieldPath:        "id",
			AcceptableValues: []string{"link-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.Attributes["parent_team_id"] != "parent-1" {
		t.Fatalf("parent_team_id = %q, want parent-1", generator.Resources[0].InstanceState.Attributes["parent_team_id"])
	}
}

func TestTeamHierarchyLinksInitResourcesFiltersByParentTeam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/team-hierarchy-links" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("filter[parent_team]"); got != "parent-1" {
			t.Fatalf("filter[parent_team] = %q, want parent-1", got)
		}
		_, _ = fmt.Fprint(w, teamHierarchyLinksListResponseJSON(teamHierarchyLinkJSON("link-1", "parent-1", "sub-1")))
	}))
	defer server.Close()

	generator := newTeamHierarchyLinksTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "team_hierarchy_links",
			FieldPath:        "parent_team_id",
			AcceptableValues: []string{"parent-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
}

func newTeamHierarchyLinksTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *TeamHierarchyLinksGenerator {
	return &TeamHierarchyLinksGenerator{
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

func teamHierarchyLinksListResponseJSON(links ...string) string {
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(links, ","))
}

func teamHierarchyLink(id string, parentTeamID string, subTeamID string) datadogV2.TeamHierarchyLink {
	parentTeam := datadogV2.NewTeamHierarchyLinkTeam(parentTeamID, datadogV2.TEAMTYPE_TEAM)
	subTeam := datadogV2.NewTeamHierarchyLinkTeam(subTeamID, datadogV2.TEAMTYPE_TEAM)
	relationships := datadogV2.NewTeamHierarchyLinkRelationships(
		*datadogV2.NewTeamHierarchyLinkTeamRelationship(*parentTeam),
		*datadogV2.NewTeamHierarchyLinkTeamRelationship(*subTeam),
	)
	teamHierarchyLink := datadogV2.NewTeamHierarchyLinkWithDefaults()
	teamHierarchyLink.SetId(id)
	teamHierarchyLink.SetRelationships(*relationships)
	return *teamHierarchyLink
}

func teamHierarchyLinkJSON(id string, parentTeamID string, subTeamID string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"team_hierarchy_links\",\"attributes\":{\"created_at\":\"2026-01-01T00:00:00Z\",\"provisioned_by\":\"user@example.com\"},\"relationships\":{\"parent_team\":{\"data\":{\"id\":%q,\"type\":\"team\"}},\"sub_team\":{\"data\":{\"id\":%q,\"type\":\"team\"}}}}",
		id,
		parentTeamID,
		subTeamID,
	)
}
