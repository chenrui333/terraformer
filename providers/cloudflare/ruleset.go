// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type RulesetGenerator struct {
	CloudflareService
}

func (g *RulesetGenerator) appendRulesetResources(ctx context.Context, api *cf.API, rc *cf.ResourceContainer, scopeType string) error {
	rulesets, err := api.ListRulesets(ctx, rc, cf.ListRulesetsParams{})
	if err != nil {
		return fmt.Errorf("%s/%s: %w", scopeType, rc.Identifier, err)
	}
	for _, ruleset := range rulesets {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			ruleset.ID,
			cloudflareResourceName(scopeType, rc.Identifier, ruleset.Name, ruleset.ID),
			"cloudflare_ruleset",
			"cloudflare",
			accessScopeAttributes(scopeType, rc.Identifier),
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}

func (g *RulesetGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	if g.accountID() != "" {
		if err := g.appendRulesetResources(ctx, api, cf.AccountIdentifier(g.accountID()), "accounts"); err != nil {
			return err
		}
	}

	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		if err := g.appendRulesetResources(ctx, api, cf.ZoneIdentifier(zone.ID), "zones"); err != nil {
			return err
		}
	}
	return nil
}
