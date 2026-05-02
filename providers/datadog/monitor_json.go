// SPDX-License-Identifier: Apache-2.0

//nolint:gosec // lint triage: legacy provider/API/security baseline is tracked in #175.
package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// MonitorJSONAllowEmptyValues ...
	MonitorJSONAllowEmptyValues = []string{}
)

// MonitorJSONGenerator ...
type MonitorJSONGenerator struct {
	DatadogService
}

func (g *MonitorJSONGenerator) createResources(monitors []datadogV1.Monitor) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, monitor := range monitors {
		if monitor.GetType() == datadogV1.MONITORTYPE_SYNTHETICS_ALERT {
			continue
		}
		resourceName := strconv.FormatInt(monitor.GetId(), 10)
		resources = append(resources, g.createResource(resourceName))
	}

	return resources
}

func (g *MonitorJSONGenerator) createResource(monitorID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		monitorID,
		fmt.Sprintf("monitor_json_%s", monitorID),
		"datadog_monitor_json",
		"datadog",
		MonitorJSONAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each monitor create 1 TerraformResource using datadog_monitor_json.
// Need Monitor ID as ID for terraform resource.
func (g *MonitorJSONGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV1.NewMonitorsApi(datadogClient)

	optionalParams := datadogV1.NewListMonitorsOptionalParameters()
	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("monitor_json") {
			hasIDFilter = true
			for _, value := range filter.AcceptableValues {
				i, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return err
				}

				monitor, httpResp, err := api.GetMonitor(auth, i)
				if httpResp != nil && httpResp.Body != nil {
					_ = httpResp.Body.Close()
				}
				if err != nil {
					return err
				}
				if monitor.GetType() == datadogV1.MONITORTYPE_SYNTHETICS_ALERT {
					continue
				}

				resources = append(resources, g.createResource(strconv.FormatInt(monitor.GetId(), 10)))
			}
		}
		if filter.FieldPath == "tags" && filter.IsApplicable("monitor_json") {
			optionalParams.WithMonitorTags(strings.Join(filter.AcceptableValues, ","))
		}
	}

	if hasIDFilter || len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	var monitors []datadogV1.Monitor
	pageSize := int32(1000)
	pageNumber := int64(0)
	for {
		resp, httpResp, err := api.ListMonitors(auth, *optionalParams.
			WithPageSize(pageSize).
			WithPage(pageNumber))
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			return err
		}

		if len(resp) == 0 || int32(len(resp)) < pageSize {
			monitors = append(monitors, resp...)
			break
		}

		monitors = append(monitors, resp...)
		pageNumber++
	}

	g.Resources = g.createResources(monitors)
	return nil
}

// PostRefreshCleanup filters datadog_monitor_json resources after provider refresh.
// The provider stores monitor fields inside the top-level monitor JSON string, so
// Terraformer's generic tag filter cannot see monitor tags without parsing it.
func (g *MonitorJSONGenerator) PostRefreshCleanup() {
	if len(g.Filter) == 0 {
		return
	}

	resources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		if g.postRefreshFiltersMatch(resource) && !terraformutils.ContainsResource(resources, resource) {
			resources = append(resources, resource)
		}
	}
	g.Resources = resources
}

func (g *MonitorJSONGenerator) postRefreshFiltersMatch(resource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" {
			continue
		}
		if filter.FieldPath == "tags" && filter.IsApplicable("monitor_json") {
			if !monitorJSONTagsFilter(resource, filter) {
				return false
			}
			continue
		}
		if !filter.Filter(resource) {
			return false
		}
	}
	return true
}

func monitorJSONTagsFilter(resource terraformutils.Resource, filter terraformutils.ResourceFilter) bool {
	monitor, ok := monitorJSONAttributes(resource)
	if !ok {
		return false
	}
	if filter.AcceptableValues == nil {
		return terraformutils.WalkAndCheckField("tags", monitor)
	}

	values := terraformutils.WalkAndGet("tags", monitor)
	for _, value := range values {
		for _, acceptableValue := range filter.AcceptableValues {
			if value == acceptableValue {
				return true
			}
		}
	}
	return false
}

func monitorJSONAttributes(resource terraformutils.Resource) (map[string]interface{}, bool) {
	monitorJSON := resource.InstanceState.Attributes["monitor"]
	if monitorJSON == "" && resource.Item != nil {
		monitor, ok := resource.Item["monitor"].(string)
		if ok {
			monitorJSON = monitor
		}
	}
	if monitorJSON == "" {
		return nil, false
	}

	monitor := map[string]interface{}{}
	if err := json.Unmarshal([]byte(monitorJSON), &monitor); err != nil {
		return nil, false
	}
	return monitor, true
}
