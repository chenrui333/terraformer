// SPDX-License-Identifier: Apache-2.0

package pagerduty

import (
	"github.com/chenrui333/terraformer/terraformutils"
	pagerduty "github.com/heimweh/go-pagerduty/pagerduty"
)

type EscalationPolicyGenerator struct {
	PagerDutyService
}

func (g *EscalationPolicyGenerator) createEscalationPolicyResources(client *pagerduty.Client) error {
	var offset = 0
	options := pagerduty.ListEscalationPoliciesOptions{}
	for {
		options.Offset = offset
		resp, _, err := client.EscalationPolicies.List(&options)
		if err != nil {
			return err
		}

		for _, policy := range resp.EscalationPolicies {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				policy.ID,
				policy.Name,
				"pagerduty_escalation_policy",
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

func (g *EscalationPolicyGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	funcs := []func(*pagerduty.Client) error{
		g.createEscalationPolicyResources,
	}

	for _, f := range funcs {
		err := f(client)
		if err != nil {
			return err
		}
	}

	return nil
}
