// SPDX-License-Identifier: Apache-2.0

package pagerduty

import (
	"github.com/chenrui333/terraformer/terraformutils"
	pagerduty "github.com/heimweh/go-pagerduty/pagerduty"
)

type BusinessServiceGenerator struct {
	PagerDutyService
}

func (g *BusinessServiceGenerator) createBusinessServiceResources(client *pagerduty.Client) error {
	resp, _, err := client.BusinessServices.List()
	if err != nil {
		return err
	}

	for _, service := range resp.BusinessServices {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			service.ID,
			service.Name,
			"pagerduty_business_service",
			g.ProviderName,
			[]string{},
		))
	}

	return nil
}

func (g *BusinessServiceGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	funcs := []func(*pagerduty.Client) error{
		g.createBusinessServiceResources,
	}

	for _, f := range funcs {
		err := f(client)
		if err != nil {
			return err
		}
	}

	return nil
}
