// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/dataproc/v1"
	"google.golang.org/api/option"
)

func TestCreateClusterResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	dataprocService, err := dataproc.NewService(ctx, option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}

	_, err = (DataprocGenerator{}).createClusterResources(ctx, dataprocService.Projects.Regions.Clusters.List("project", "region"))
	if err == nil {
		t.Fatal("expected dataproc cluster list error")
	}
	if !strings.Contains(err.Error(), "list dataproc clusters") {
		t.Fatalf("expected wrapped dataproc cluster list error, got %q", err)
	}
}
