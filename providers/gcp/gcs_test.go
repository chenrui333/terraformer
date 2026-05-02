// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
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

func TestCreateBucketsResourcesReturnsIAMPolicyError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/b":
			fmt.Fprint(w, "{\"items\":[{\"name\":\"test-bucket\"}]}")
		case "/b/test-bucket/iam":
			http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
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
		t.Fatal("expected gcs bucket IAM policy error")
	}
	if !strings.Contains(err.Error(), "get gcs bucket IAM policy for test-bucket") {
		t.Fatalf("expected wrapped gcs bucket IAM policy error, got %q", err)
	}
}

func TestCreateNotificationResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	gcsService, err := storage.NewService(ctx, option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}

	_, err = (&GcsGenerator{}).createNotificationResources(gcsService, &storage.Bucket{Name: "test-bucket"})
	if err == nil {
		t.Fatal("expected gcs notification list error")
	}
	if !strings.Contains(err.Error(), "list gcs notifications for test-bucket") {
		t.Fatalf("expected wrapped gcs notification list error, got %q", err)
	}
}
