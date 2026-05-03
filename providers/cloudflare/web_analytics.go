// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type WebAnalyticsGenerator struct {
	CloudflareService
}

func (g *WebAnalyticsGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	params := cf.ListWebAnalyticsSitesParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		sites, info, err := api.ListWebAnalyticsSites(ctx, account, params)
		if err != nil {
			return err
		}
		for _, site := range sites {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				site.SiteTag,
				cloudflareResourceName(account.Identifier, site.SiteTag),
				"cloudflare_web_analytics_site",
				"cloudflare",
				map[string]string{"account_id": account.Identifier, "site_tag": site.SiteTag},
				[]string{},
				map[string]interface{}{},
			))
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}
