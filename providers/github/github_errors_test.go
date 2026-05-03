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
	provider := GithubProvider{
		owner:          "old-owner",
		token:          "old-token",
		baseURL:        "https://github.example.com/api/v3/",
		appID:          123,
		installationID: 456,
		pem:            "old-pem",
	}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing owner error")
	}
	if !strings.Contains(err.Error(), "owner is required") {
		t.Fatalf("Init error = %q, want missing owner", err)
	}
	if provider.owner != "" {
		t.Fatalf("owner = %q, want empty after failed init", provider.owner)
	}
	if provider.token != "" {
		t.Fatalf("token = %q, want empty after failed init", provider.token)
	}
	if provider.baseURL != githubDefaultURL {
		t.Fatalf("baseURL = %q, want %q after failed init", provider.baseURL, githubDefaultURL)
	}
	if provider.appID != 0 || provider.installationID != 0 || provider.pem != "" {
		t.Fatalf("app auth = (%d, %d, %q), want cleared after failed init", provider.appID, provider.installationID, provider.pem)
	}
}

func TestGithubProviderInitUsesEnvTokenForEmptyTokenArg(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")
	t.Setenv("GITHUB_APP_ID", "")
	t.Setenv("GITHUB_APP_INSTALLATION_ID", "")
	t.Setenv("GITHUB_APP_PEM_FILE", "")
	var provider GithubProvider

	if err := provider.Init([]string{"test-org", "", ""}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if provider.token != "env-token" {
		t.Fatalf("token = %q, want env-token", provider.token)
	}
	if provider.baseURL != githubDefaultURL {
		t.Fatalf("baseURL = %q, want %q", provider.baseURL, githubDefaultURL)
	}
}

func TestGithubProviderInitReturnsTokenErrorForEmptyTokenArgWithoutEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITHUB_APP_ID", "")
	t.Setenv("GITHUB_APP_INSTALLATION_ID", "")
	t.Setenv("GITHUB_APP_PEM_FILE", "")
	provider := GithubProvider{
		owner:          "old-owner",
		token:          "old-token",
		baseURL:        "https://github.example.com/api/v3/",
		appID:          123,
		installationID: 456,
		pem:            "old-pem",
	}

	err := provider.Init([]string{"test-org", "", ""})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), "token requirement") {
		t.Fatalf("Init error = %q, want token requirement", err)
	}
	if provider.owner != "" {
		t.Fatalf("owner = %q, want empty after failed init", provider.owner)
	}
	if provider.token != "" {
		t.Fatalf("token = %q, want empty after failed init", provider.token)
	}
	if provider.baseURL != githubDefaultURL {
		t.Fatalf("baseURL = %q, want %q after failed init", provider.baseURL, githubDefaultURL)
	}
	if provider.appID != 0 || provider.installationID != 0 || provider.pem != "" {
		t.Fatalf("app auth = (%d, %d, %q), want cleared after failed init", provider.appID, provider.installationID, provider.pem)
	}
}

func TestGithubProviderInitClearsStaleOptionalAuthConfig(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")
	t.Setenv("GITHUB_APP_ID", "")
	t.Setenv("GITHUB_APP_INSTALLATION_ID", "")
	t.Setenv("GITHUB_APP_PEM_FILE", "")
	provider := GithubProvider{
		token:          "old-token",
		baseURL:        "https://github.example.com/api/v3/",
		appID:          123,
		installationID: 456,
		pem:            "old-pem",
	}

	if err := provider.Init([]string{"test-org"}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if provider.token != "env-token" {
		t.Fatalf("token = %q, want env-token", provider.token)
	}
	if provider.baseURL != githubDefaultURL {
		t.Fatalf("baseURL = %q, want %q", provider.baseURL, githubDefaultURL)
	}
	if provider.appID != 0 || provider.installationID != 0 || provider.pem != "" {
		t.Fatalf("app auth = (%d, %d, %q), want cleared", provider.appID, provider.installationID, provider.pem)
	}
}

func TestGithubServiceCreateClientReturnsAppAuthError(t *testing.T) {
	service := &GithubService{}
	service.SetArgs(map[string]interface{}{
		"base_url":        githubDefaultURL,
		"app_id":          int64(123),
		"installation_id": int64(456),
		"pem":             "not a pem",
		"token":           "",
	})

	client, err := service.createClient()
	if err == nil {
		t.Fatal("expected GitHub app auth setup error")
	}
	if client != nil {
		t.Fatalf("client = %v, want nil", client)
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
