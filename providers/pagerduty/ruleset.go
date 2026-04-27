// SPDX-License-Identifier: Apache-2.0

package pagerduty

import (
	"github.com/chenrui333/terraformer/terraformutils"
	pagerduty "github.com/heimweh/go-pagerduty/pagerduty"
)

type RulesetGenerator struct {
	PagerDutyService
}

func (g *RulesetGenerator) createRulesetResources(client *pagerduty.Client) error {
	resp, _, err := client.Rulesets.List()
	if err != nil {
		return err
	}

	for _, ruleset := range resp.Rulesets {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			ruleset.ID,
			ruleset.Name,
			"pagerduty_ruleset",
			g.ProviderName,
			[]string{},
		))
	}

	return nil
}

// golangci-lint says this function isn't used anywhere. Do we need it? Commenting it out to make the linter happy

func (g *RulesetGenerator) createRulesetRuleResources(client *pagerduty.Client) error {
	resp, _, err := client.Rulesets.List()
	if err != nil {
		return err
	}

	for _, ruleset := range resp.Rulesets {
		rules, _, err := client.Rulesets.ListRules(ruleset.ID)
		if err != nil {
			return err
		}

		for _, rule := range rules.Rules {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				rule.ID,
				rule.ID,
				"pagerduty_ruleset_rule",
				g.ProviderName,
				map[string]string{
					"ruleset": ruleset.ID,
				},
				[]string{},
				map[string]interface{}{},
			))
		}
	}

	return nil
}

func (g *RulesetGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	funcs := []func(*pagerduty.Client) error{
		g.createRulesetResources,
		g.createRulesetRuleResources,
	}

	for _, f := range funcs {
		err := f(client)
		if err != nil {
			return err
		}
	}

	return nil
}
