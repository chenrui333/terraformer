// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSyntheticsGlobalVariableInitResourcesReturnsFilteredIDError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/synthetics/variables/global-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		http.Error(w, "{\"errors\":[\"service unavailable\"]}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	generator := &SyntheticsGlobalVariableGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": datadog.NewAPIClient(config),
				},
				Filter: []terraformutils.ResourceFilter{
					{
						ServiceName:      "synthetics_global_variable",
						FieldPath:        "id",
						AcceptableValues: []string{"global-1"},
					},
				},
			},
		},
	}

	err := generator.InitResources()
	if err == nil {
		t.Fatal("expected filtered synthetics global variable error")
	}
	if !strings.Contains(err.Error(), "get datadog synthetics global variable global-1") {
		t.Fatalf("InitResources error = %q, want wrapped global variable error", err)
	}
}

func TestSyntheticsGlobalVariableInitResourcesOmitsFilteredIDSucceeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request for omitted synthetics global variable: %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	generator := &SyntheticsGlobalVariableGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": datadog.NewAPIClient(config),
				},
				Filter: []terraformutils.ResourceFilter{
					{
						ServiceName:      "synthetics_global_variable",
						FieldPath:        "id",
						AcceptableValues: nil,
					},
				},
			},
		},
	}

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error for omitted synthetics global variable ID: %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("Resources length = %d, want 0", len(generator.Resources))
	}
}
