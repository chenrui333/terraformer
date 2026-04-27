// SPDX-License-Identifier: Apache-2.0

package pagerduty

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	pagerduty "github.com/heimweh/go-pagerduty/pagerduty"
)

type ScheduleGenerator struct {
	PagerDutyService
}

func (g *ScheduleGenerator) createScheduleResources(client *pagerduty.Client) error {
	var offset = 0
	options := pagerduty.ListSchedulesOptions{}
	for {
		options.Offset = offset
		resp, _, err := client.Schedules.List(&options)
		if err != nil {
			return err
		}

		for _, schedule := range resp.Schedules {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				schedule.ID,
				fmt.Sprintf("schedule_%s", schedule.Name),
				"pagerduty_schedule",
				g.ProviderName,
				[]string{},
			))
		}
		if !resp.More {
			break
		}

		offset += resp.Limit
	}

	return nil
}

func (g *ScheduleGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	funcs := []func(*pagerduty.Client) error{
		g.createScheduleResources,
	}

	for _, f := range funcs {
		err := f(client)
		if err != nil {
			return err
		}
	}

	return nil
}
