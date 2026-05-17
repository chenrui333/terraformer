// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const datadogDowntimeSchedulePageLimit = int64(100)

var (
	// DowntimeScheduleAllowEmptyValues ...
	DowntimeScheduleAllowEmptyValues = []string{}
)

// DowntimeScheduleGenerator ...
type DowntimeScheduleGenerator struct {
	DatadogService
}

func (g *DowntimeScheduleGenerator) createResource(downtime datadogV2.DowntimeResponseData) (terraformutils.Resource, error) {
	downtimeID := downtime.GetId()
	if downtimeID == "" {
		return terraformutils.Resource{}, fmt.Errorf("downtime schedule missing id")
	}

	return terraformutils.NewSimpleResource(
		downtimeID,
		fmt.Sprintf("downtime_schedule_%s", downtimeID),
		"datadog_downtime_schedule",
		"datadog",
		DowntimeScheduleAllowEmptyValues,
	), nil
}

func (g *DowntimeScheduleGenerator) createResources(downtimes []datadogV2.DowntimeResponseData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, downtime := range downtimes {
		resource, err := g.createResource(downtime)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each Datadog V2 downtime schedule create 1 TerraformResource.
func (g *DowntimeScheduleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewDowntimesApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	downtimes, err := listDowntimeSchedules(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(downtimes)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *DowntimeScheduleGenerator) filteredResources(auth context.Context, api *datadogV2.DowntimesApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("downtime_schedule") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			downtime, err := getDowntimeSchedule(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(downtime)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getDowntimeSchedule(auth context.Context, api *datadogV2.DowntimesApi, downtimeID string) (datadogV2.DowntimeResponseData, error) {
	resp, httpResp, err := api.GetDowntime(auth, downtimeID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return datadogV2.DowntimeResponseData{}, err
	}
	if data, ok := resp.GetDataOk(); ok {
		return *data, nil
	}
	return datadogV2.DowntimeResponseData{}, fmt.Errorf("downtime schedule %q not found", downtimeID)
}

func listDowntimeSchedules(auth context.Context, api *datadogV2.DowntimesApi) ([]datadogV2.DowntimeResponseData, error) {
	downtimes := []datadogV2.DowntimeResponseData{}
	offset := int64(0)

	for {
		optionalParams := datadogV2.NewListDowntimesOptionalParameters().
			WithPageOffset(offset).
			WithPageLimit(datadogDowntimeSchedulePageLimit)

		resp, httpResp, err := api.ListDowntimes(auth, *optionalParams)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		pageDowntimes := resp.GetData()
		downtimes = append(downtimes, pageDowntimes...)

		if len(pageDowntimes) == 0 || len(pageDowntimes) < int(datadogDowntimeSchedulePageLimit) {
			break
		}
		meta := resp.GetMeta()
		page := meta.GetPage()
		if total := page.GetTotalFilteredCount(); total > 0 && len(downtimes) >= int(total) {
			break
		}
		offset += int64(len(pageDowntimes))
	}

	return downtimes, nil
}
