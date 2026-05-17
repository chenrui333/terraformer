// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestDowntimeScheduleCreateResource(t *testing.T) {
	downtime := datadogV2.NewDowntimeResponseDataWithDefaults()
	downtime.SetId("downtime-123")

	generator := &DowntimeScheduleGenerator{}
	resource, err := generator.createResource(*downtime)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "downtime-123" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "downtime-123")
	}
	if resource.ResourceName != "tfer--downtime_schedule_downtime-123" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--downtime_schedule_downtime-123")
	}
	if resource.InstanceInfo.Type != "datadog_downtime_schedule" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_downtime_schedule")
	}
}

func TestDowntimeScheduleCreateResourceMissingID(t *testing.T) {
	generator := &DowntimeScheduleGenerator{}
	_, err := generator.createResource(datadogV2.DowntimeResponseData{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestDowntimeScheduleCreateResources(t *testing.T) {
	firstDowntime := datadogV2.NewDowntimeResponseDataWithDefaults()
	firstDowntime.SetId("downtime-1")
	secondDowntime := datadogV2.NewDowntimeResponseDataWithDefaults()
	secondDowntime.SetId("downtime-2")

	generator := &DowntimeScheduleGenerator{}
	resources, err := generator.createResources([]datadogV2.DowntimeResponseData{*firstDowntime, *secondDowntime})
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}

	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(resources), 2)
	}
	if resources[0].ResourceName == resources[1].ResourceName {
		t.Fatalf("resource names should be unique, got %q", resources[0].ResourceName)
	}
}

func TestDowntimeScheduleInitResourcesList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/downtime" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("page[offset]"); got != "0" {
			t.Errorf("page[offset] query = %q, want %q", got, "0")
		}
		if got := r.URL.Query().Get("page[limit]"); got != fmt.Sprint(datadogDowntimeSchedulePageLimit) {
			t.Errorf("page[limit] query = %q, want %q", got, fmt.Sprint(datadogDowntimeSchedulePageLimit))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"data\":[{\"id\":\"downtime-1\",\"type\":\"downtime\"}],\"meta\":{\"page\":{\"total_filtered_count\":1}}}"))
	}))
	t.Cleanup(server.Close)

	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	generator := &DowntimeScheduleGenerator{
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
	if generator.Resources[0].InstanceState.ID != "downtime-1" {
		t.Fatalf("resource ID = %q, want %q", generator.Resources[0].InstanceState.ID, "downtime-1")
	}
}
