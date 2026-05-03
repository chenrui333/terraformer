package opal

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	opalsdk "github.com/opalsecurity/opal-go"
)

type OnCallScheduleGenerator struct {
	OpalService
}

func (g *OnCallScheduleGenerator) createResources(onCallSchedules []opalsdk.OnCallSchedule) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	countByName := make(map[string]int)

	for _, onCallSchedule := range onCallSchedules {
		resourceID, err := opalRequiredStringPtr("opal_on_call_schedule", "on_call_schedule_id", onCallSchedule.OnCallScheduleId)
		if err != nil {
			return nil, err
		}
		name := opalUniqueResourceName(opalResourceDisplayName(onCallSchedule.Name, resourceID), countByName)

		resources = append(resources, terraformutils.NewSimpleResource(
			resourceID,
			name,
			"opal_on_call_schedule",
			"opal",
			[]string{},
		))
	}

	return resources, nil
}

func (g *OnCallScheduleGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return fmt.Errorf("unable to list opal on call schedules: %w", err)
	}

	onCallSchedules, _, err := client.OnCallSchedulesAPI.GetOnCallSchedules(context.TODO()).Execute()
	if err != nil {
		return fmt.Errorf("unable to list opal on call schedules: %w", err)
	}

	resources, err := g.createResources(onCallSchedules.OnCallSchedules)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, resources...)

	return nil
}
