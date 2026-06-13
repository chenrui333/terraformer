// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	githubAPI "github.com/google/go-github/v88/github"
)

func TestGithubProviderServiceRegistration(t *testing.T) {
	var provider GithubProvider

	if got := provider.GetName(); got != "github" {
		t.Fatalf("GetName() = %q, want github", got)
	}

	services := provider.GetSupportedService()
	for _, name := range []string{
		"members",
		"organization",
		"organization_blocks",
		"organization_projects",
		"organization_webhooks",
		"repositories",
		"teams",
		"user_ssh_keys",
	} {
		if services[name] == nil {
			t.Fatalf("supported services missing %q", name)
		}
	}

	if err := provider.Init([]string{"test-org", "test-token"}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if err := provider.InitService("repositories", false); err != nil {
		t.Fatalf("InitService returned error: %v", err)
	}
	args := provider.Service.GetArgs()
	if got := args["owner"]; got != "test-org" {
		t.Fatalf("service owner arg = %q, want test-org", got)
	}
	if got := args["token"]; got != "test-token" {
		t.Fatalf("service token arg = %q, want test-token", got)
	}
	if got := args["base_url"]; got != githubDefaultURL {
		t.Fatalf("service base_url arg = %q, want %q", got, githubDefaultURL)
	}

	if err := provider.InitService("missing", false); err == nil {
		t.Fatal("expected unsupported service error")
	}
}

func TestGithubProviderInitEmptyBaseURLUsesPublicDefault(t *testing.T) {
	var provider GithubProvider

	if err := provider.Init([]string{"test-org", "test-token", ""}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if provider.baseURL != githubDefaultURL {
		t.Fatalf("baseURL = %q, want %q", provider.baseURL, githubDefaultURL)
	}
}

func TestGithubServiceCreateEnterpriseClientUsesNormalizedBaseURL(t *testing.T) {
	var seenPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
			http.Error(w, "unexpected method", http.StatusBadRequest)
			return
		}
		seenPath = r.URL.Path
		if r.URL.Path != "/api/v3/orgs/test-org/repos" {
			t.Errorf("path = %s, want /api/v3/orgs/test-org/repos", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer token", got)
			http.Error(w, "unexpected auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	service := &GithubService{}
	service.SetArgs(map[string]interface{}{
		"base_url":        strings.TrimRight(server.URL, "/"),
		"app_id":          int64(0),
		"installation_id": int64(0),
		"pem":             "",
		"token":           "test-token",
	})

	client, err := service.createClient()
	if err != nil {
		t.Fatalf("createClient returned error: %v", err)
	}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), "test-org", &githubAPI.RepositoryListByOrgOptions{
		ListOptions: githubAPI.ListOptions{PerPage: 100},
	})
	if err != nil {
		t.Fatalf("ListByOrg returned error: %v", err)
	}
	if len(repos) != 0 {
		t.Fatalf("repos = %d, want 0", len(repos))
	}
	if seenPath != "/api/v3/orgs/test-org/repos" {
		t.Fatalf("seen path = %q, want enterprise API path", seenPath)
	}
}

func TestGithubServiceCreateEnterpriseClientRejectsInvalidBaseURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		errorText string
	}{
		{
			name:      "missing scheme",
			baseURL:   "github.example.com/api/v3",
			errorText: "scheme must be http or https",
		},
		{
			name:      "credentials",
			baseURL:   "https://user:pass@github.example.com/api/v3",
			errorText: "credentials are not allowed",
		},
		{
			name:      "query",
			baseURL:   "https://github.example.com/api/v3?token=secret",
			errorText: "query parameters are not allowed",
		},
		{
			name:      "fragment",
			baseURL:   "https://github.example.com/api/v3#fragment",
			errorText: "fragments are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &GithubService{}
			service.SetArgs(map[string]interface{}{
				"base_url":        tt.baseURL,
				"app_id":          int64(0),
				"installation_id": int64(0),
				"pem":             "",
				"token":           "test-token",
			})

			client, err := service.createClient()
			if err == nil {
				t.Fatal("expected invalid base_url error")
			}
			if client != nil {
				t.Fatalf("client = %v, want nil", client)
			}
			if !strings.Contains(err.Error(), tt.errorText) {
				t.Fatalf("error = %q, want %q", err, tt.errorText)
			}
		})
	}
}
