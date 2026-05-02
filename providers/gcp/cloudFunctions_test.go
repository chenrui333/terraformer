// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/cloudfunctions/v2"
	"google.golang.org/api/option"
)

func TestCreateCloudFunctionsResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	cloudFunctionsService := newTestCloudFunctionsService(ctx, t, server.URL+"/")
	_, err := (CloudFunctionsGenerator{}).createCloudFunctionsResources(ctx, cloudFunctionsService.Projects.Locations.Functions.List("projects/test-project/locations/us-central1"))
	if err == nil {
		t.Fatal("expected cloud functions gen1 list error")
	}
	if !strings.Contains(err.Error(), "list cloud functions gen1") {
		t.Fatalf("expected wrapped cloud functions gen1 list error, got %q", err)
	}
}

func TestCreateCloudFunctions2ndGenResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	cloudFunctionsService := newTestCloudFunctionsService(ctx, t, server.URL+"/")
	_, err := (CloudFunctionsGenerator{}).createCloudFunctions2ndGenResources(ctx, cloudFunctionsService.Projects.Locations.Functions.List("projects/test-project/locations/us-central1"))
	if err == nil {
		t.Fatal("expected cloud functions gen2 list error")
	}
	if !strings.Contains(err.Error(), "list cloud functions gen2") {
		t.Fatalf("expected wrapped cloud functions gen2 list error, got %q", err)
	}
}

func newTestCloudFunctionsService(ctx context.Context, t *testing.T, endpoint string) *cloudfunctions.Service {
	t.Helper()

	cloudFunctionsService, err := cloudfunctions.NewService(ctx, option.WithEndpoint(endpoint), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return cloudFunctionsService
}
