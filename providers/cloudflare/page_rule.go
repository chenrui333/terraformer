// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

type PageRulesGenerator struct {
	CloudflareService
}

func (g *PageRulesGenerator) createPageRules(api *cf.API, zoneID string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	pageRules, err := api.ListPageRules(context.Background(), zoneID)
	if err != nil {
		return resources, err
	}

	for _, pageRule := range pageRules {
		resources = append(resources, terraformutils.NewResource(
			pageRule.ID,
			pageRule.ID,
			"cloudflare_page_rule",
			"cloudflare",
			map[string]string{
				"zone_id": zoneID,
			},
			[]string{},
			map[string]interface{}{},
		))
	}

	return resources, nil
}

func (g *PageRulesGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	zones, err := api.ListZones(context.Background())
	if err != nil {
		return err
	}

	for _, zone := range zones {
		resources, err := g.createPageRules(api, zone.ID)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	return nil
}
