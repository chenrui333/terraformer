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

type EscalationPoliciesGenerator struct {
	PagerDutyService
}

func (g *EscalationPoliciesGenerator) createEscalationPoliciesResources(client *pagerduty.Client) ([]*pagerduty.EscalationPolicy, error) {
	limit, offset := 100, 0
	opt := &pagerduty.ListEscalationPoliciesOptions{Limit: limit, Offset: offset}
	escalationPoliciesResponse := &pagerduty.ListEscalationPoliciesResponse{}

	for {
		escalationPoliciesResponse, _, err := client.EscalationPolicies.List(opt)

		if err != nil {
			return nil, err
		}

		for _, escalationPolicy := range escalationPoliciesResponse.EscalationPolicies {
			g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
				escalationPolicy.ID,
				escalationPolicy.ID,
				"pagerduty_escalation_policy",
				"pagerduty",
				[]string{}))
		}

		if !escalationPoliciesResponse.More {
			break
		}

		opt.Offset = opt.Offset + opt.Limit
	}
	return escalationPoliciesResponse.EscalationPolicies, nil
}

// InitResources generates TerraformResources from PagerDuty API,
func (g *EscalationPoliciesGenerator) InitResources() error {
	client, err := pagerduty.NewClient(&pagerduty.Config{Token:g.Args["api_key"].(string)})

	if err != nil {
		return err
	}

	escalationPolicys, err := g.createEscalationPoliciesResources(client)
	if err != nil {
		return err
	}

	print(escalationPolicys)

	return nil
}
