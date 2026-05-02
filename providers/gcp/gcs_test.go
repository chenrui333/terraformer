// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

func TestCreateBucketsResourcesReturnsBucketListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	gcsService, err := storage.NewService(ctx, option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}

	g := GcsGenerator{}
	g.SetArgs(map[string]interface{}{"project": "test-project"})

	_, err = g.createBucketsResources(ctx, gcsService)
	if err == nil {
		t.Fatal("expected gcs bucket list error")
	}
	if !strings.Contains(err.Error(), "list gcs buckets") {
		t.Fatalf("expected wrapped gcs bucket list error, got %q", err)
	}
}
