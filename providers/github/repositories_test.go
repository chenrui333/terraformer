// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	githubAPI "github.com/google/go-github/v88/github"
)

func TestRepositoryChildResourcesPaginateAndMap(t *testing.T) {
	ctx := context.Background()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, ok := githubTestPage(t, r)
		if !ok {
			http.Error(w, "unexpected pagination query", http.StatusBadRequest)
			return
		}

		switch r.URL.Path {
		case "/repos/test-org/test-repo/hooks":
			switch page {
			case 1:
				writeGitHubTestPage(w, r, server.URL, 2, `[{"id":101}]`)
			case 2:
				writeGitHubTestPage(w, r, server.URL, 0, `[{"id":102}]`)
			default:
				unexpectedGitHubPage(t, w, r, page)
			}
		case "/repos/test-org/test-repo/branches":
			switch page {
			case 1:
				writeGitHubTestPage(w, r, server.URL, 2, `[{"name":"main","protected":true}]`)
			case 2:
				writeGitHubTestPage(w, r, server.URL, 0, `[{"name":"dev","protected":false},{"name":"release","protected":true}]`)
			default:
				unexpectedGitHubPage(t, w, r, page)
			}
		case "/repos/test-org/test-repo/collaborators":
			switch page {
			case 1:
				writeGitHubTestPage(w, r, server.URL, 2, `[{"login":"alice"}]`)
			case 2:
				writeGitHubTestPage(w, r, server.URL, 0, `[{"login":"bob"}]`)
			default:
				unexpectedGitHubPage(t, w, r, page)
			}
		case "/repos/test-org/test-repo/keys":
			switch page {
			case 1:
				writeGitHubTestPage(w, r, server.URL, 2, `[{"id":201,"title":"deploy-main"}]`)
			case 2:
				writeGitHubTestPage(w, r, server.URL, 0, `[{"id":202,"title":"deploy-release"}]`)
			default:
				unexpectedGitHubPage(t, w, r, page)
			}
		default:
			unexpectedGitHubRequest(t, w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestGitHubClient(t, server)
	g := RepositoriesGenerator{}
	g.SetArgs(map[string]interface{}{"owner": "test-org"})
	repoName := "test-repo"
	repo := &githubAPI.Repository{Name: &repoName}

	hooks, err := g.createRepositoryWebhookResources(ctx, client, repo)
	if err != nil {
		t.Fatalf("createRepositoryWebhookResources returned error: %v", err)
	}
	assertGithubResources(t, hooks, []githubResourceWant{
		{id: "101", name: "tfer--test-repo_101", typ: "github_repository_webhook"},
		{id: "102", name: "tfer--test-repo_102", typ: "github_repository_webhook"},
	})
	if got := hooks[0].InstanceState.Attributes["repository"]; got != "test-repo" {
		t.Fatalf("webhook repository attribute = %q, want test-repo", got)
	}

	branches, err := g.createRepositoryBranchProtectionResources(ctx, client, repo)
	if err != nil {
		t.Fatalf("createRepositoryBranchProtectionResources returned error: %v", err)
	}
	assertGithubResources(t, branches, []githubResourceWant{
		{id: "test-repo:main", name: "tfer--test-repo_main", typ: "github_branch_protection"},
		{id: "test-repo:release", name: "tfer--test-repo_release", typ: "github_branch_protection"},
	})

	collaborators, err := g.createRepositoryCollaboratorResources(ctx, client, repo)
	if err != nil {
		t.Fatalf("createRepositoryCollaboratorResources returned error: %v", err)
	}
	assertGithubResources(t, collaborators, []githubResourceWant{
		{id: "test-repo:alice", name: "tfer--test-repo-003A-alice", typ: "github_repository_collaborator"},
		{id: "test-repo:bob", name: "tfer--test-repo-003A-bob", typ: "github_repository_collaborator"},
	})

	keys, err := g.createRepositoryDeployKeyResources(ctx, client, repo)
	if err != nil {
		t.Fatalf("createRepositoryDeployKeyResources returned error: %v", err)
	}
	assertGithubResources(t, keys, []githubResourceWant{
		{id: "test-repo:201", name: "tfer--test-repo-003A-deploy-main", typ: "github_repository_deploy_key"},
		{id: "test-repo:202", name: "tfer--test-repo-003A-deploy-release", typ: "github_repository_deploy_key"},
	})
}

func TestCreateRepositoryWebhookResourcesReturnsSecondPageError(t *testing.T) {
	ctx := context.Background()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, ok := githubTestPage(t, r)
		if !ok {
			http.Error(w, "unexpected pagination query", http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/repos/test-org/test-repo/hooks" {
			unexpectedGitHubRequest(t, w, r)
			return
		}

		switch page {
		case 1:
			writeGitHubTestPage(w, r, server.URL, 2, `[{"id":101}]`)
		case 2:
			http.Error(w, `{"message":"service unavailable"}`, http.StatusServiceUnavailable)
		default:
			unexpectedGitHubPage(t, w, r, page)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestGitHubClient(t, server)
	g := RepositoriesGenerator{}
	g.SetArgs(map[string]interface{}{"owner": "test-org"})

	repoName := "test-repo"
	_, err := g.createRepositoryWebhookResources(ctx, client, &githubAPI.Repository{Name: &repoName})
	if err == nil {
		t.Fatal("expected second page webhook error")
	}
	if !strings.Contains(err.Error(), "list github repository webhooks for test-repo") {
		t.Fatalf("error = %q, want repository webhook context", err)
	}
}

func TestCreateMembershipsResourcesHandlesEmptyPage(t *testing.T) {
	ctx := context.Background()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, ok := githubTestPage(t, r)
		if !ok {
			http.Error(w, "unexpected pagination query", http.StatusBadRequest)
			return
		}
		if page != 1 {
			unexpectedGitHubPage(t, w, r, page)
			return
		}
		if r.URL.Path != "/orgs/test-org/members" {
			unexpectedGitHubRequest(t, w, r)
			return
		}
		writeGitHubTestPage(w, r, server.URL, 0, `[]`)
	}))
	t.Cleanup(server.Close)

	client := newTestGitHubClient(t, server)

	resources, err := createMembershipsResources(ctx, client, "test-org")
	if err != nil {
		t.Fatalf("createMembershipsResources returned error: %v", err)
	}
	if len(resources) != 0 {
		t.Fatalf("resources = %d, want 0", len(resources))
	}
}

type githubResourceWant struct {
	id   string
	name string
	typ  string
}

func assertGithubResources(t *testing.T, resources []terraformutils.Resource, wants []githubResourceWant) {
	t.Helper()

	if len(resources) != len(wants) {
		t.Fatalf("resources = %d, want %d", len(resources), len(wants))
	}
	for i, want := range wants {
		resource := resources[i]
		if resource.InstanceState.ID != want.id {
			t.Fatalf("resources[%d].InstanceState.ID = %q, want %q", i, resource.InstanceState.ID, want.id)
		}
		if resource.ResourceName != want.name {
			t.Fatalf("resources[%d].ResourceName = %q, want %q", i, resource.ResourceName, want.name)
		}
		if resource.InstanceInfo.Type != want.typ {
			t.Fatalf("resources[%d].InstanceInfo.Type = %q, want %q", i, resource.InstanceInfo.Type, want.typ)
		}
	}
}

func githubTestPage(t *testing.T, r *http.Request) (int, bool) {
	if got := r.URL.Query().Get("per_page"); got != "100" {
		t.Errorf("request %s per_page = %q, want 100", r.URL.Path, got)
		return 0, false
	}
	pageValue := r.URL.Query().Get("page")
	if pageValue == "" {
		return 1, true
	}
	page, err := strconv.Atoi(pageValue)
	if err != nil {
		t.Errorf("request %s page = %q, want integer: %v", r.URL.Path, pageValue, err)
		return 0, false
	}
	return page, true
}

func writeGitHubTestPage(w http.ResponseWriter, r *http.Request, serverURL string, nextPage int, body string) {
	w.Header().Set("Content-Type", "application/json")
	if nextPage != 0 {
		nextURL := *r.URL
		nextQuery := nextURL.Query()
		nextQuery.Set("page", strconv.Itoa(nextPage))
		nextURL.RawQuery = nextQuery.Encode()
		w.Header().Set("Link", fmt.Sprintf("<%s%s>; rel=\"next\"", serverURL, nextURL.String()))
	}
	_, _ = w.Write([]byte(body))
}

func unexpectedGitHubRequest(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Errorf("unexpected %s request to %s", r.Method, r.URL.String())
	http.Error(w, "unexpected request", http.StatusNotFound)
}

func unexpectedGitHubPage(t *testing.T, w http.ResponseWriter, r *http.Request, page int) {
	t.Errorf("unexpected page %d for %s", page, r.URL.String())
	http.Error(w, "unexpected page", http.StatusNotFound)
}
