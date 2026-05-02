// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func TestGeneratedComputeResourcesReturnRegionalListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	computeService := newTestComputeService(ctx, t, server.URL+"/")
	_, err := (AddressesGenerator{}).createResources(ctx, computeService.Addresses.List("test-project", "us-central1"))
	if err == nil {
		t.Fatal("expected address list error")
	}
	if !strings.Contains(err.Error(), "list addresses") {
		t.Fatalf("expected wrapped address list error, got %q", err)
	}
}

func TestGeneratedComputeResourcesReturnZonalListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	computeService := newTestComputeService(ctx, t, server.URL+"/")
	_, err := (DisksGenerator{}).createResources(ctx, computeService.Disks.List("test-project", "us-central1-a"), "us-central1-a")
	if err == nil {
		t.Fatal("expected disk list error")
	}
	if !strings.Contains(err.Error(), "list disks") {
		t.Fatalf("expected wrapped disk list error, got %q", err)
	}
}

func newTestComputeService(ctx context.Context, t *testing.T, endpoint string) *compute.Service {
	t.Helper()

	computeService, err := compute.NewService(ctx, option.WithEndpoint(endpoint), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return computeService
}
