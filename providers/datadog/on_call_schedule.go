// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// OnCallScheduleAllowEmptyValues ...
	OnCallScheduleAllowEmptyValues = []string{"teams"}
)

// OnCallScheduleGenerator ...
type OnCallScheduleGenerator struct {
	DatadogService
}

func (g *OnCallScheduleGenerator) createResource(onCallSchedule datadogV2.Schedule) (terraformutils.Resource, error) {
	data := onCallSchedule.GetData()
	onCallScheduleID := data.GetId()
	if onCallScheduleID == "" {
		return terraformutils.Resource{}, fmt.Errorf("On-Call schedule missing id")
	}

	return terraformutils.NewSimpleResource(
		onCallScheduleID,
		fmt.Sprintf("on_call_schedule_%s", onCallScheduleID),
		"datadog_on_call_schedule",
		"datadog",
		OnCallScheduleAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each On-Call schedule create 1 TerraformResource.
// Need On-Call Schedule ID as ID for terraform resource.
func (g *OnCallScheduleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewOnCallApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	g.Resources = []terraformutils.Resource{}
	return nil
}

func (g *OnCallScheduleGenerator) filteredResources(auth context.Context, api *datadogV2.OnCallApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if !filter.IsApplicable("on_call_schedule") || filter.FieldPath != "id" {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			onCallSchedule, err := getOnCallSchedule(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(onCallSchedule)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getOnCallSchedule(auth context.Context, api *datadogV2.OnCallApi, scheduleID string) (datadogV2.Schedule, error) {
	include := "layers,layers.members.user"
	onCallSchedule, httpResp, err := api.GetOnCallSchedule(auth, scheduleID, datadogV2.GetOnCallScheduleOptionalParameters{Include: &include})
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}
	if err != nil {
		return datadogV2.Schedule{}, err
	}

	data := onCallSchedule.GetData()
	if data.GetId() == "" {
		data.SetId(scheduleID)
		onCallSchedule.SetData(data)
	}
	return onCallSchedule, nil
}
