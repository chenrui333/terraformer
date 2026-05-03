// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type RulesetGenerator struct {
	CloudflareService
}

func (g *RulesetGenerator) appendRulesetResources(ctx context.Context, api *cf.API, rc *cf.ResourceContainer, scopeType string) error {
	rulesets, err := listRulesets(ctx, api, rc)
	if err != nil {
		return fmt.Errorf("%s/%s: %w", scopeType, rc.Identifier, err)
	}
	for _, ruleset := range rulesets {
		if ruleset.Kind == string(cf.RulesetKindManaged) {
			continue
		}
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

func listRulesets(ctx context.Context, api *cf.API, rc *cf.ResourceContainer) ([]cf.Ruleset, error) {
	var rulesets []cf.Ruleset
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/%s/%s/rulesets?%s", rc.Level, rc.Identifier, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageRulesets []cf.Ruleset
		if err := json.Unmarshal(response.Result, &pageRulesets); err != nil {
			return nil, err
		}
		rulesets = append(rulesets, pageRulesets...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return rulesets, nil
}

func (g *RulesetGenerator) InitResources() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

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
