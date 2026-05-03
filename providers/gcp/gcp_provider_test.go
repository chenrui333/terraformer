// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/compute/v1"
)

func TestGCPProviderInitRequiresRegion(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")

	provider := GCPProvider{
		projectName:  "old-project",
		region:       compute.Region{Name: "old-region"},
		providerType: "beta",
	}
	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing region error")
	}
	if !strings.Contains(err.Error(), "gcp region must be provided") {
		t.Fatalf("Init error = %q, want missing region", err)
	}
	if provider.projectName != "" {
		t.Fatalf("projectName = %q, want empty after failed init", provider.projectName)
	}
	if provider.region.Name != "" {
		t.Fatalf("region.Name = %q, want empty after failed init", provider.region.Name)
	}
	if provider.providerType != "" {
		t.Fatalf("providerType = %q, want empty after failed init", provider.providerType)
	}
}

func TestGCPProviderInitAllowsDefaultProviderType(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")

	provider := GCPProvider{}
	if err := provider.Init([]string{"global"}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if provider.GetName() != "google" {
		t.Fatalf("GetName() = %q, want %q", provider.GetName(), "google")
	}
}

func TestGCPProviderInitClearsProviderTypeWhenOmitted(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")

	provider := GCPProvider{}
	if err := provider.Init([]string{"global", "test-project", "beta"}); err != nil {
		t.Fatalf("Init with beta provider returned error: %v", err)
	}
	if provider.GetName() != "google-beta" {
		t.Fatalf("GetName() = %q, want %q", provider.GetName(), "google-beta")
	}

	if err := provider.Init([]string{"global"}); err != nil {
		t.Fatalf("Init without provider type returned error: %v", err)
	}
	if provider.GetName() != "google" {
		t.Fatalf("GetName() = %q, want %q", provider.GetName(), "google")
	}
}

func TestGCPProviderInitReturnsNonGlobalRegionLookupError(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"region lookup failed\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	originalNewComputeService := newComputeService
	t.Cleanup(func() {
		newComputeService = originalNewComputeService
	})
	newComputeService = func(ctx context.Context) (*compute.Service, error) {
		return newTestComputeService(ctx, t, server.URL+"/"), nil
	}

	provider := GCPProvider{
		projectName:  "old-project",
		region:       compute.Region{Name: "old-region"},
		providerType: "beta",
	}
	err := provider.Init([]string{"us-west1"})
	if err == nil {
		t.Fatal("expected region lookup error")
	}
	want := `get GCP region "us-west1" for project "test-project"`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Init error = %q, want it to contain %q", err, want)
	}
	if provider.projectName != "" {
		t.Fatalf("projectName = %q, want empty after failed region lookup", provider.projectName)
	}
	if provider.region.Name != "" {
		t.Fatalf("region.Name = %q, want empty after failed region lookup", provider.region.Name)
	}
	if provider.providerType != "" {
		t.Fatalf("providerType = %q, want empty after failed region lookup", provider.providerType)
	}
}
