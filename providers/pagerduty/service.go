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

type ServicesGenerator struct {
	PagerDutyService
}

func (g *ServicesGenerator) createServicesResources(client *pagerduty.Client) ([]*pagerduty.Service, error) {
	limit, offset := 100, 0
	opt := &pagerduty.ListServicesOptions{Limit: limit, Offset: offset}
	servicesResponse := &pagerduty.ListServicesResponse{}

	for {
		servicesResponse, _, err := client.Services.List(opt)

		if err != nil {
			return nil, err
		}

		for _, service := range servicesResponse.Services {
			g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
				service.ID,
				service.ID,
				"pagerduty_service",
				"pagerduty",
				[]string{}))
		}

		if !servicesResponse.More {
			break
		}

		opt.Offset = opt.Offset + opt.Limit
	}
	return servicesResponse.Services, nil
}

// InitResources generates TerraformResources from PagerDuty API,
func (g *ServicesGenerator) InitResources() error {
	client, err := pagerduty.NewClient(&pagerduty.Config{Token: g.Args["api_key"].(string)})

	if err != nil {
		return err
	}

	services, err := g.createServicesResources(client)
	if err != nil {
		return err
	}

	print(services)

	return nil
}
