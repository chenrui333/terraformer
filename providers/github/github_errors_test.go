// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	githubAPI "github.com/google/go-github/v35/github"
)

func TestCreateMembershipsResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := newErrorGitHubServer(t)
	client := newTestGitHubClient(t, server)

	_, err := createMembershipsResources(ctx, client, "test-org")
	if err == nil {
		t.Fatal("expected github members list error")
	}
	if !strings.Contains(err.Error(), "list github members for test-org") {
		t.Fatalf("expected wrapped github members list error, got %q", err)
	}
}

func TestGithubProviderInitRequiresOwner(t *testing.T) {
	var provider GithubProvider

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing owner error")
	}
	if !strings.Contains(err.Error(), "owner is required") {
		t.Fatalf("Init error = %q, want missing owner", err)
	}
}

func TestCreateRepositoryWebhookResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := newErrorGitHubServer(t)
	client := newTestGitHubClient(t, server)

	g := RepositoriesGenerator{}
	g.SetArgs(map[string]interface{}{"owner": "test-org"})

	_, err := g.createRepositoryWebhookResources(ctx, client, &githubAPI.Repository{Name: githubAPI.String("test-repo")})
	if err == nil {
		t.Fatal("expected github repository webhook list error")
	}
	if !strings.Contains(err.Error(), "list github repository webhooks for test-repo") {
		t.Fatalf("expected wrapped github repository webhook list error, got %q", err)
	}
}

func TestCreateTeamsResourcesReturnsNestedListError(t *testing.T) {
	ctx := context.Background()
	server := newErrorGitHubServer(t)
	client := newTestGitHubClient(t, server)

	g := TeamsGenerator{}
	g.SetArgs(map[string]interface{}{"owner": "test-org"})

	_, err := g.createTeamsResources(ctx, []*githubAPI.Team{{Slug: githubAPI.String("test-team")}}, client)
	if err == nil {
		t.Fatal("expected github team member list error")
	}
	if !strings.Contains(err.Error(), "list github team members for test-team") {
		t.Fatalf("expected wrapped github team member list error, got %q", err)
	}
}

func newErrorGitHubServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"message\":\"service unavailable\"}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	return server
}

func newTestGitHubClient(t *testing.T, server *httptest.Server) *githubAPI.Client {
	t.Helper()

	client := githubAPI.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = baseURL
	client.UploadURL = baseURL
	return client
}
