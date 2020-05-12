// Copyright 2020 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pagerduty

import (
	"github.com/GoogleCloudPlatform/terraformer/terraform_utils"
	"github.com/heimweh/go-pagerduty/pagerduty"
)

type SchedulesGenerator struct {
	PagerDutyService
}

func (g *SchedulesGenerator) createSchedulesResources(client *pagerduty.Client) ([]*pagerduty.Schedule, error) {
	limit, offset := 100, 0
	opt := &pagerduty.ListSchedulesOptions{Limit: limit, Offset: offset}
	schedulesResponse := &pagerduty.ListSchedulesResponse{}

	for {
		schedulesResponse, _, err := client.Schedules.List(opt)

		if err != nil {
			return nil, err
		}

		for _, schedule := range schedulesResponse.Schedules {
			g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
				schedule.ID,
				schedule.ID,
				"pagerduty_schedule",
				"pagerduty",
				[]string{}))
		}

		if !schedulesResponse.More {
			break
		}

		opt.Offset = opt.Offset + opt.Limit
	}
	return schedulesResponse.Schedules, nil
}

// InitResources generates TerraformResources from PagerDuty API,
func (g *SchedulesGenerator) InitResources() error {
	client, err := pagerduty.NewClient(&pagerduty.Config{Token: g.Args["api_key"].(string)})

	if err != nil {
		return err
	}

	schedules, err := g.createSchedulesResources(client)
	if err != nil {
		return err
	}

	print(schedules)

	return nil
}
