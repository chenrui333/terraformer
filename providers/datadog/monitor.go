// SPDX-License-Identifier: Apache-2.0

//nolint:gosec // lint triage: legacy provider/API/security baseline is tracked in #175.
package datadog

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// MonitorAllowEmptyValues ...
	MonitorAllowEmptyValues = []string{"tags.", "message"}
)

// MonitorGenerator ...
type MonitorGenerator struct {
	DatadogService
}

func (g *MonitorGenerator) createResources(monitors []datadogV1.Monitor) []terraformutils.Resource {
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

func (g *MonitorGenerator) createResource(monitorID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		monitorID,
		fmt.Sprintf("monitor_%s", monitorID),
		"datadog_monitor",
		"datadog",
		MonitorAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each monitor create 1 TerraformResource.
// Need Monitor ID as ID for terraform resource
func (g *MonitorGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV1.NewMonitorsApi(datadogClient)

	optionalParams := datadogV1.NewListMonitorsOptionalParameters()
	resources := []terraformutils.Resource{}
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("monitor") {
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

				resources = append(resources, g.createResource(strconv.FormatInt(monitor.GetId(), 10)))
			}
		}
		if filter.FieldPath == "tags" && filter.IsApplicable("monitor") {
			optionalParams.WithMonitorTags(strings.Join(filter.AcceptableValues, ","))
		}
	}

	if len(resources) > 0 {
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
