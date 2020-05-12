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

type ExtensionsGenerator struct {
	PagerDutyService
}

func (g *ExtensionsGenerator) createExtensionsResources(client *pagerduty.Client) ([]*pagerduty.Extension, error) {
	opt := &pagerduty.ListExtensionsOptions{}
	extensionsResponse, _, err := client.Extensions.List(opt)

	if err != nil {
		return nil, err
	}

	for _, extension := range extensionsResponse.Extensions {
		g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
			extension.ID,
			extension.ID,
			"pagerduty_extension",
			"pagerduty",
			[]string{}))
	}

	return extensionsResponse.Extensions, nil
}

// InitResources generates TerraformResources from PagerDuty API,
func (g *ExtensionsGenerator) InitResources() error {
	client, err := pagerduty.NewClient(&pagerduty.Config{Token: g.Args["api_key"].(string)})

	if err != nil {
		return err
	}

	extensions, err := g.createExtensionsResources(client)
	if err != nil {
		return err
	}

	print(extensions)

	return nil
}
