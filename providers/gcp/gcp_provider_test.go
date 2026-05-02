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

	provider := GCPProvider{}
	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing region error")
	}
	if !strings.Contains(err.Error(), "gcp region must be provided") {
		t.Fatalf("Init error = %q, want missing region", err)
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

	provider := GCPProvider{}
	err := provider.Init([]string{"us-west1"})
	if err == nil {
		t.Fatal("expected region lookup error")
	}
	want := `get GCP region "us-west1" for project "test-project"`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Init error = %q, want it to contain %q", err, want)
	}
}
