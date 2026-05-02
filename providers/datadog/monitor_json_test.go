// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestMonitorJSONCreateResource(t *testing.T) {
	generator := &MonitorJSONGenerator{}
	resource := generator.createResource("12345")

	if resource.InstanceState.ID != "12345" {
		t.Fatalf("expected resource ID 12345, got %s", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--monitor_json_12345" {
		t.Fatalf("expected resource name tfer--monitor_json_12345, got %s", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_monitor_json" {
		t.Fatalf("expected resource type datadog_monitor_json, got %s", resource.InstanceInfo.Type)
	}
}

func TestMonitorJSONCreateResourcesSkipsSyntheticsAlerts(t *testing.T) {
	metricMonitor := datadogV1.NewMonitorWithDefaults()
	metricMonitor.SetId(1001)
	metricMonitor.SetType(datadogV1.MONITORTYPE_METRIC_ALERT)

	syntheticsMonitor := datadogV1.NewMonitorWithDefaults()
	syntheticsMonitor.SetId(1002)
	syntheticsMonitor.SetType(datadogV1.MONITORTYPE_SYNTHETICS_ALERT)

	generator := &MonitorJSONGenerator{}
	resources := generator.createResources([]datadogV1.Monitor{*metricMonitor, *syntheticsMonitor})

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "1001" {
		t.Fatalf("expected resource ID 1001, got %s", resources[0].InstanceState.ID)
	}
}

func TestMonitorJSONInitResourcesKeepsIDFilterTerminalWhenAllMatchesAreSkipped(t *testing.T) {
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/monitor/1002":
			_, _ = fmt.Fprint(w, "{\"id\":1002,\"type\":\"synthetics alert\",\"query\":\"synthetics query\"}")
		case "/api/v1/monitor":
			_, _ = fmt.Fprint(w, "[]")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	generator := &MonitorJSONGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": datadog.NewAPIClient(config),
				},
				Filter: []terraformutils.ResourceFilter{
					{
						ServiceName:      "monitor_json",
						FieldPath:        "id",
						AcceptableValues: []string{"1002"},
					},
				},
			},
		},
	}

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("expected no resources, got %d", len(generator.Resources))
	}
	for _, path := range requestedPaths {
		if path == "/api/v1/monitor" {
			t.Fatalf("expected id filter to avoid full monitor list, got requests %v", requestedPaths)
		}
	}
}

func TestMonitorJSONPostRefreshCleanupMatchesTagsInsideMonitorJSON(t *testing.T) {
	matching := (&MonitorJSONGenerator{}).createResource("1001")
	matching.InstanceState.Attributes["monitor"] = "{\"tags\":[\"env:prod\",\"team:core\"],\"query\":\"avg:system.cpu.user{*} > 10\",\"type\":\"metric alert\"}"

	nonMatching := (&MonitorJSONGenerator{}).createResource("1002")
	nonMatching.InstanceState.Attributes["monitor"] = "{\"tags\":[\"env:stage\"],\"query\":\"avg:system.cpu.user{*} > 10\",\"type\":\"metric alert\"}"

	generator := &MonitorJSONGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{matching, nonMatching},
				Filter: []terraformutils.ResourceFilter{
					{
						FieldPath:        "tags",
						AcceptableValues: []string{"env:prod"},
					},
				},
			},
		},
	}

	generator.PostRefreshCleanup()

	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "1001" {
		t.Fatalf("expected resource ID 1001, got %s", generator.Resources[0].InstanceState.ID)
	}
}
