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

type TeamsGenerator struct {
	PagerDutyService
}

func (g *TeamsGenerator) createTeamsResources(client *pagerduty.Client) ([]*pagerduty.Team, error) {
	limit, offset := 100, 0
	opt := &pagerduty.ListTeamsOptions{Limit: limit, Offset: offset}
	teamsResponse := &pagerduty.ListTeamsResponse{}

	for {
		teamsResponse, _, err := client.Teams.List(opt)

		if err != nil {
			return nil, err
		}

		for _, team := range teamsResponse.Teams {
			g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
				team.ID,
				team.ID,
				"pagerduty_team",
				"pagerduty",
				[]string{}))
		}

		if !teamsResponse.More {
			break
		}

		opt.Offset = opt.Offset + opt.Limit
	}
	return teamsResponse.Teams, nil
}

// InitResources generates TerraformResources from PagerDuty API,
func (g *TeamsGenerator) InitResources() error {
	client, err := pagerduty.NewClient(&pagerduty.Config{Token:g.Args["api_key"].(string)})

	if err != nil {
		return err
	}

	teams, err := g.createTeamsResources(client)
	if err != nil {
		return err
	}

	print(teams)

	return nil
}
