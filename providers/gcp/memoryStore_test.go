// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/redis/v1"
)

func TestCreateMemoryStoreResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	redisService, err := redis.NewService(ctx, option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}

	_, err = (MemoryStoreGenerator{}).createResources(ctx, redisService.Projects.Locations.Instances.List("projects/test-project/locations/us-central1"))
	if err == nil {
		t.Fatal("expected redis instance list error")
	}
	if !strings.Contains(err.Error(), "list redis instances") {
		t.Fatalf("expected wrapped redis instance list error, got %q", err)
	}
}
