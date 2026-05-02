// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cloudscheduler "google.golang.org/api/cloudscheduler/v1beta1"
	"google.golang.org/api/option"
)

func TestCreateSchedulerJobResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	schedulerService, err := cloudscheduler.NewService(ctx, option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}

	_, err = (SchedulerJobsGenerator{}).createResources(ctx, schedulerService.Projects.Locations.Jobs.List("projects/test-project/locations/us-central1"))
	if err == nil {
		t.Fatal("expected scheduler job list error")
	}
	if !strings.Contains(err.Error(), "list scheduler jobs") {
		t.Fatalf("expected wrapped scheduler job list error, got %q", err)
	}
}
