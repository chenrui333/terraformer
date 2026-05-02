// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gitlabAPI "gitlab.com/gitlab-org/api/client-go"
)

func TestCreateGroupsReturnsGroupGetError(t *testing.T) {
	ctx := context.Background()
	client := newErrorGitLabClient(t)

	_, err := createGroups(ctx, client, "test-group")
	if err == nil {
		t.Fatal("expected gitlab group get error")
	}
	if !strings.Contains(err.Error(), "get gitlab group test-group") {
		t.Fatalf("expected wrapped gitlab group get error, got %q", err)
	}
}

func TestCreateProjectsReturnsProjectListError(t *testing.T) {
	ctx := context.Background()
	client := newErrorGitLabClient(t)

	_, err := createProjects(ctx, client, "test-group")
	if err == nil {
		t.Fatal("expected gitlab project list error")
	}
	if !strings.Contains(err.Error(), "list gitlab projects for test-group") {
		t.Fatalf("expected wrapped gitlab project list error, got %q", err)
	}
}

func TestCreateProjectVariablesReturnsListError(t *testing.T) {
	ctx := context.Background()
	client := newErrorGitLabClient(t)

	_, err := createProjectVariables(ctx, client, &gitlabAPI.Project{ID: 123})
	if err == nil {
		t.Fatal("expected gitlab project variable list error")
	}
	if !strings.Contains(err.Error(), "list gitlab project variables for 123") {
		t.Fatalf("expected wrapped gitlab project variable list error, got %q", err)
	}
}

func newErrorGitLabClient(t *testing.T) *gitlabAPI.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"message\":\"service unavailable\"}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	client, err := gitlabAPI.NewClient("token", gitlabAPI.WithBaseURL(server.URL), gitlabAPI.WithHTTPClient(server.Client()), gitlabAPI.WithoutRetries())
	if err != nil {
		t.Fatal(err)
	}
	return client
}
