// SPDX-License-Identifier: Apache-2.0

package pagerduty

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	pagerduty "github.com/heimweh/go-pagerduty/pagerduty"
)

type ServiceGenerator struct {
	PagerDutyService
}

func (g *ServiceGenerator) createServiceResources(client *pagerduty.Client) error {
	var offset = 0
	options := pagerduty.ListServicesOptions{}
	for {
		options.Offset = offset
		resp, _, err := client.Services.List(&options)
		if err != nil {
			return err
		}

		for _, service := range resp.Services {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				service.ID,
				fmt.Sprintf("service_%s", service.Name),
				"pagerduty_service",
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

func (g *ServiceGenerator) createServiceEventRuleResources(client *pagerduty.Client) error {
	var offset = 0
	options := pagerduty.ListServicesOptions{}
	optionsEventRules := pagerduty.ListServiceEventRuleOptions{}
	for {
		options.Offset = offset
		optionsEventRules.Offset = offset
		resp, _, err := client.Services.List(&options)
		if err != nil {
			return err
		}

		for _, service := range resp.Services {
			rules, _, err := client.Services.ListEventRules(service.ID, &optionsEventRules)

			if err != nil {
				return err
			}

			for _, rule := range rules.EventRules {
				g.Resources = append(g.Resources, terraformutils.NewResource(
					rule.ID,
					fmt.Sprintf("%s_%s", service.Name, rule.ID),
					"pagerduty_service_event_rule",
					g.ProviderName,
					map[string]string{
						"service": service.ID,
					},
					[]string{},
					map[string]interface{}{},
				))
			}
		}

		if !resp.More {
			break
		}
		offset += resp.Limit
	}
	return nil
}

func (g *ServiceGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	funcs := []func(*pagerduty.Client) error{
		g.createServiceResources,
		g.createServiceEventRuleResources,
	}

	for _, f := range funcs {
		err := f(client)
		if err != nil {
			return err
		}
	}

	return nil
}
