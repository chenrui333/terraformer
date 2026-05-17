// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSyntheticsConcurrencyCapCreateResource(t *testing.T) {
	attrs := datadogV2.NewOnDemandConcurrencyCapAttributesWithDefaults()
	attrs.SetOnDemandConcurrencyCap(7)
	data := datadogV2.NewOnDemandConcurrencyCapWithDefaults()
	data.SetAttributes(*attrs)
	resp := datadogV2.NewOnDemandConcurrencyCapResponseWithDefaults()
	resp.SetData(*data)

	generator := &SyntheticsConcurrencyCapGenerator{}
	resource, err := generator.createResource(*resp)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != syntheticsConcurrencyCapID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, syntheticsConcurrencyCapID)
	}
	if resource.ResourceName != "tfer--synthetics_concurrency_cap" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--synthetics_concurrency_cap")
	}
	if resource.InstanceInfo.Type != "datadog_synthetics_concurrency_cap" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_synthetics_concurrency_cap")
	}
	if resource.InstanceState.Attributes["on_demand_concurrency_cap"] != "7" {
		t.Fatalf("on_demand_concurrency_cap = %q, want %q", resource.InstanceState.Attributes["on_demand_concurrency_cap"], "7")
	}
}

func TestSyntheticsConcurrencyCapCreateResourceMissingValue(t *testing.T) {
	generator := &SyntheticsConcurrencyCapGenerator{}
	_, err := generator.createResource(datadogV2.OnDemandConcurrencyCapResponse{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing cap error")
	}
}

func TestSyntheticsConcurrencyCapInitResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/synthetics/settings/on_demand_concurrency_cap" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"data\":{\"type\":\"on_demand_concurrency_cap\",\"attributes\":{\"on_demand_concurrency_cap\":3}}}"))
	}))
	t.Cleanup(server.Close)

	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	generator := &SyntheticsConcurrencyCapGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": datadog.NewAPIClient(config),
				},
			},
		},
	}

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resource count = %d, want %d", len(generator.Resources), 1)
	}
	if generator.Resources[0].InstanceState.ID != syntheticsConcurrencyCapID {
		t.Fatalf("resource ID = %q, want %q", generator.Resources[0].InstanceState.ID, syntheticsConcurrencyCapID)
	}
}

func TestSyntheticsConcurrencyCapInitResourcesNormalizesIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/synthetics/settings/on_demand_concurrency_cap" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"data\":{\"type\":\"on_demand_concurrency_cap\",\"attributes\":{\"on_demand_concurrency_cap\":3}}}"))
	}))
	t.Cleanup(server.Close)

	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	generator := &SyntheticsConcurrencyCapGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": datadog.NewAPIClient(config),
				},
				Filter: []terraformutils.ResourceFilter{
					{
						ServiceName:      syntheticsConcurrencyCapServiceName,
						FieldPath:        "id",
						AcceptableValues: []string{"this"},
					},
				},
			},
		},
	}

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if got := generator.Filter[0].AcceptableValues; len(got) != 1 || got[0] != syntheticsConcurrencyCapID {
		t.Fatalf("rewritten id filter = %v, want [%s]", got, syntheticsConcurrencyCapID)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resource count = %d, want %d", len(generator.Resources), 1)
	}
	if !generator.Filter[0].Filter(generator.Resources[0]) {
		t.Fatal("expected normalized ID filter to keep synthetics concurrency cap resource")
	}
}

func TestSyntheticsConcurrencyCapInitResourcesDoesNotNormalizeGlobalIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/synthetics/settings/on_demand_concurrency_cap" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"data\":{\"type\":\"on_demand_concurrency_cap\",\"attributes\":{\"on_demand_concurrency_cap\":3}}}"))
	}))
	t.Cleanup(server.Close)

	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	generator := &SyntheticsConcurrencyCapGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": datadog.NewAPIClient(config),
				},
				Filter: []terraformutils.ResourceFilter{
					{
						FieldPath:        "id",
						AcceptableValues: []string{"some-monitor-id"},
					},
				},
			},
		},
	}

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if got := generator.Filter[0].AcceptableValues; len(got) != 1 || got[0] != "some-monitor-id" {
		t.Fatalf("global id filter was rewritten to %v", got)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resource count before cleanup = %d, want %d", len(generator.Resources), 1)
	}

	generator.InitialCleanup()
	if len(generator.Resources) != 0 {
		t.Fatalf("resource count after cleanup = %d, want %d", len(generator.Resources), 0)
	}
}
