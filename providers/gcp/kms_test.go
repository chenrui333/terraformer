// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"
)

func TestCreateKmsRingResourcesReturnsKeyRingListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	kmsService := newTestKmsService(ctx, t, server.URL+"/")
	_, err := (KmsGenerator{}).createKmsRingResources(ctx, kmsService.Projects.Locations.KeyRings.List("projects/test-project/locations/global"), kmsService)
	if err == nil {
		t.Fatal("expected kms key ring list error")
	}
	if !strings.Contains(err.Error(), "list kms key rings") {
		t.Fatalf("expected wrapped kms key ring list error, got %q", err)
	}
}

func TestCreateKmsRingResourcesReturnsCryptoKeyListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/cryptoKeys") {
			http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"keyRings\":[{\"name\":\"projects/test-project/locations/global/keyRings/test-ring\"}]}"))
	}))
	t.Cleanup(server.Close)

	kmsService := newTestKmsService(ctx, t, server.URL+"/")
	g := KmsGenerator{}
	g.SetArgs(map[string]interface{}{"project": "test-project"})

	_, err := g.createKmsRingResources(ctx, kmsService.Projects.Locations.KeyRings.List("projects/test-project/locations/global"), kmsService)
	if err == nil {
		t.Fatal("expected kms crypto key list error")
	}
	if !strings.Contains(err.Error(), "list kms crypto keys") {
		t.Fatalf("expected wrapped kms crypto key list error, got %q", err)
	}
}

func newTestKmsService(ctx context.Context, t *testing.T, endpoint string) *cloudkms.Service {
	t.Helper()

	kmsService, err := cloudkms.NewService(ctx, option.WithEndpoint(endpoint), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return kmsService
}
