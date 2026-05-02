// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
)

func TestCreateDatasetsReturnsDatasetListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	bigQueryService := newTestBigQueryService(ctx, t, server.URL+"/")
	_, err := (BigQueryGenerator{}).createDatasets(ctx, bigQueryService.Datasets.List("test-project"), bigQueryService)
	if err == nil {
		t.Fatal("expected bigquery dataset list error")
	}
	if !strings.Contains(err.Error(), "list bigquery datasets") {
		t.Fatalf("expected wrapped bigquery dataset list error, got %q", err)
	}
}

func TestCreateDatasetsReturnsTableListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tables") {
			http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"datasets\":[{\"id\":\"test-project:test_dataset\"}]}"))
	}))
	t.Cleanup(server.Close)

	bigQueryService := newTestBigQueryService(ctx, t, server.URL+"/")
	g := BigQueryGenerator{}
	g.SetArgs(map[string]interface{}{"project": "test-project"})

	_, err := g.createDatasets(ctx, bigQueryService.Datasets.List("test-project"), bigQueryService)
	if err == nil {
		t.Fatal("expected bigquery table list error")
	}
	if !strings.Contains(err.Error(), "list bigquery tables for test_dataset") {
		t.Fatalf("expected wrapped bigquery table list error, got %q", err)
	}
}

func newTestBigQueryService(ctx context.Context, t *testing.T, endpoint string) *bigquery.Service {
	t.Helper()

	bigQueryService, err := bigquery.NewService(ctx, option.WithEndpoint(endpoint), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return bigQueryService
}
