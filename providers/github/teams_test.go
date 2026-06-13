// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	githubAPI "github.com/google/go-github/v88/github"
)

func TestTeamChildResourcesPaginateAndMap(t *testing.T) {
	ctx := context.Background()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, ok := githubTestPage(t, r)
		if !ok {
			http.Error(w, "unexpected pagination query", http.StatusBadRequest)
			return
		}

		switch r.URL.Path {
		case "/orgs/test-org/teams/platform/members":
			switch page {
			case 1:
				writeGitHubTestPage(w, r, server.URL, 2, `[{"login":"alice"}]`)
			case 2:
				writeGitHubTestPage(w, r, server.URL, 0, `[{"login":"bob"}]`)
			default:
				unexpectedGitHubPage(t, w, r, page)
			}
		case "/orgs/test-org/teams/platform/repos":
			switch page {
			case 1:
				writeGitHubTestPage(w, r, server.URL, 2, `[{"name":"api"}]`)
			case 2:
				writeGitHubTestPage(w, r, server.URL, 0, `[{"name":"worker"}]`)
			default:
				unexpectedGitHubPage(t, w, r, page)
			}
		default:
			unexpectedGitHubRequest(t, w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestGitHubClient(t, server)
	g := TeamsGenerator{}
	g.SetArgs(map[string]interface{}{"owner": "test-org"})
	teamID := int64(42)
	teamName := "Platform"
	teamSlug := "platform"
	team := &githubAPI.Team{
		ID:   &teamID,
		Name: &teamName,
		Slug: &teamSlug,
	}

	members, err := g.createTeamMembersResources(ctx, team, client)
	if err != nil {
		t.Fatalf("createTeamMembersResources returned error: %v", err)
	}
	assertGithubResources(t, members, []githubResourceWant{
		{id: "42:alice", name: "tfer--Platform_alice", typ: "github_team_membership"},
		{id: "42:bob", name: "tfer--Platform_bob", typ: "github_team_membership"},
	})

	repos, err := g.createTeamRepositoriesResources(ctx, team, client)
	if err != nil {
		t.Fatalf("createTeamRepositoriesResources returned error: %v", err)
	}
	assertGithubResources(t, repos, []githubResourceWant{
		{id: "42:api", name: "tfer--Platform_api", typ: "github_team_repository"},
		{id: "42:worker", name: "tfer--Platform_worker", typ: "github_team_repository"},
	})
}
