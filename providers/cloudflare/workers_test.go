// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

func TestCloudflareWorkerResource(t *testing.T) {
	resource, ok := cloudflareWorkerResource("account-123", cloudflareWorker{
		ID:   "worker-456",
		Name: "api-worker",
	})
	if !ok {
		t.Fatal("expected Worker resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_worker" {
		t.Fatalf("resource type = %q, want cloudflare_worker", resource.InstanceInfo.Type)
	}
	if resource.InstanceState.ID != "worker-456" {
		t.Fatalf("resource ID = %q, want worker-456", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "api-worker" {
		t.Fatalf("name = %q, want api-worker", got)
	}
	if got, want := resource.ResourceName, terraformutils.TfSanitize(cloudflareResourceName("account-123", "api-worker", "worker-456")); got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/worker-456" {
		t.Fatalf("import_id = %q, want account-123/worker-456", got)
	}
	for _, key := range cloudflareWorkerComputedKeys {
		if !cloudflareResourceIgnoresKey(resource, key) {
			t.Fatalf("Worker resource should ignore computed key %q", key)
		}
	}
}

func TestCloudflareWorkerResourceSkipsMalformedWorkers(t *testing.T) {
	for name, worker := range map[string]cloudflareWorker{
		"missing id":   {Name: "api-worker"},
		"missing name": {ID: "worker-456"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := cloudflareWorkerResource("account-123", worker); ok {
				t.Fatal("expected malformed Worker to be skipped")
			}
		})
	}
	if _, ok := cloudflareWorkerResource("", cloudflareWorker{ID: "worker-456", Name: "api-worker"}); ok {
		t.Fatal("expected Worker without account ID to be skipped")
	}
}

func TestListWorkersPaginates(t *testing.T) {
	api := newCloudflareWorkersTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/workers/workers" {
			t.Fatalf("path = %q, want /accounts/account-123/workers/workers", r.URL.Path)
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			if got := r.URL.Query().Get("page"); got != "1" {
				t.Fatalf("page query = %q, want 1", got)
			}
			writeCloudflareWorkersTestResponse(t, w, []cloudflareWorker{{ID: "worker-1", Name: "api"}}, map[string]interface{}{
				"cursors": map[string]string{"after": "cursor-2"},
			})
		case "cursor-2":
			writeCloudflareWorkersTestResponse(t, w, []cloudflareWorker{{ID: "worker-2", Name: "jobs"}}, map[string]interface{}{
				"cursors": map[string]string{},
			})
		default:
			t.Fatalf("cursor query = %q, want empty or cursor-2", r.URL.Query().Get("cursor"))
		}
	}))

	workers, err := listWorkers(context.Background(), api, "account-123")
	if err != nil {
		t.Fatalf("listWorkers() error = %v", err)
	}
	if len(workers) != 2 {
		t.Fatalf("worker count = %d, want 2", len(workers))
	}
	if workers[0].ID != "worker-1" || workers[1].ID != "worker-2" {
		t.Fatalf("workers = %#v, want worker-1 and worker-2", workers)
	}
}

func TestListWorkersPaginatesPageOnlyResultInfo(t *testing.T) {
	api := newCloudflareWorkersTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/workers/workers" {
			t.Fatalf("path = %q, want /accounts/account-123/workers/workers", r.URL.Path)
		}
		if got := r.URL.Query().Get("cursor"); got != "" {
			t.Fatalf("cursor query = %q, want empty", got)
		}
		switch r.URL.Query().Get("page") {
		case "1":
			writeCloudflareWorkersTestResponse(t, w, cloudflareWorkersForTest(cloudflarePageSize), map[string]interface{}{
				"page":     1,
				"per_page": cloudflarePageSize,
			})
		case "2":
			writeCloudflareWorkersTestResponse(t, w, []cloudflareWorker{{ID: "worker-last", Name: "last"}}, map[string]interface{}{
				"page":     2,
				"per_page": cloudflarePageSize,
			})
		default:
			t.Fatalf("page query = %q, want 1 or 2", r.URL.Query().Get("page"))
		}
	}))

	workers, err := listWorkers(context.Background(), api, "account-123")
	if err != nil {
		t.Fatalf("listWorkers() error = %v", err)
	}
	if len(workers) != cloudflarePageSize+1 {
		t.Fatalf("worker count = %d, want %d", len(workers), cloudflarePageSize+1)
	}
	if workers[cloudflarePageSize].ID != "worker-last" {
		t.Fatalf("last worker = %#v, want worker-last", workers[cloudflarePageSize])
	}
}

func TestAppendWorkerResourcesHandlesEmptyAndMalformedResponses(t *testing.T) {
	api := newCloudflareWorkersTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareWorkersTestResponse(t, w, []cloudflareWorker{
			{ID: "worker-1", Name: "api"},
			{ID: "worker-2"},
			{Name: "missing-id"},
		}, nil)
	}))
	generator := &WorkersGenerator{}
	if err := generator.appendWorkerResources(context.Background(), api, "account-123"); err != nil {
		t.Fatalf("appendWorkerResources() error = %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resource count = %d, want 1", len(generator.Resources))
	}
	if got := generator.Resources[0].InstanceInfo.Type; got != "cloudflare_worker" {
		t.Fatalf("resource type = %q, want cloudflare_worker", got)
	}

	emptyAPI := newCloudflareWorkersTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCloudflareWorkersTestResponse(t, w, []cloudflareWorker{}, nil)
	}))
	generator = &WorkersGenerator{}
	if err := generator.appendWorkerResources(context.Background(), emptyAPI, "account-123"); err != nil {
		t.Fatalf("appendWorkerResources() empty response error = %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("empty response resource count = %d, want 0", len(generator.Resources))
	}
}

func TestCloudflareWorkersOptionalDiscoveryError(t *testing.T) {
	notFoundErr := cf.NewNotFoundError(&cf.Error{ErrorMessages: []string{"not found"}})
	if !cloudflareWorkersOptionalDiscoveryError(&notFoundErr) {
		t.Fatal("not found errors should be optional for Workers discovery")
	}
	authenticationErr := cf.NewAuthenticationError(&cf.Error{ErrorMessages: []string{"forbidden"}})
	if !cloudflareWorkersOptionalDiscoveryError(&authenticationErr) {
		t.Fatal("authentication errors should be optional for Workers discovery")
	}
	authorizationErr := cf.NewAuthorizationError(&cf.Error{ErrorMessages: []string{"missing permission"}})
	if !cloudflareWorkersOptionalDiscoveryError(&authorizationErr) {
		t.Fatal("authorization errors should be optional for Workers discovery")
	}
	if cloudflareWorkersOptionalDiscoveryError(errors.New("temporary Cloudflare failure")) {
		t.Fatal("generic errors should not be optional for Workers discovery")
	}
}

func TestWorkersUnsupportedResourcesMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var metadata cloudflareUnsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	seen := map[string]bool{}
	for _, resource := range metadata.Resources {
		seen[resource.Resource] = true
	}
	for _, resource := range []string{
		"cloudflare_snippet",
		"cloudflare_snippet_rules",
		"cloudflare_snippets",
		"cloudflare_worker_version",
		"cloudflare_workers_deployment",
		"cloudflare_workers_script",
		"cloudflare_workers_script_subdomain",
		"cloudflare_workflow",
	} {
		if !seen[resource] {
			t.Fatalf("unsupported metadata is missing %s", resource)
		}
	}
}

func newCloudflareWorkersTestAPI(t *testing.T, handler http.Handler) *cf.API {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	api, err := cf.NewWithAPIToken(
		"test-token",
		cf.BaseURL(server.URL),
		cf.UsingRateLimit(100000),
		cf.UsingRetryPolicy(0, 0, 0),
	)
	if err != nil {
		t.Fatalf("create Cloudflare test API: %v", err)
	}
	return api
}

func writeCloudflareWorkersTestResponse(t *testing.T, w http.ResponseWriter, result interface{}, resultInfo interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")

	payload := map[string]interface{}{
		"success": true,
		"result":  result,
	}
	if resultInfo != nil {
		payload["result_info"] = resultInfo
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode test response: %v", err)
	}
}

func cloudflareWorkersForTest(count int) []cloudflareWorker {
	workers := make([]cloudflareWorker, count)
	for i := range workers {
		workers[i] = cloudflareWorker{
			ID:   fmt.Sprintf("worker-%d", i),
			Name: fmt.Sprintf("worker-%d", i),
		}
	}
	return workers
}
