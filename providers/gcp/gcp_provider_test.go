// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"strings"
	"testing"
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
