// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type TurnstileGenerator struct {
	CloudflareService
}

func (g *TurnstileGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	params := cf.ListTurnstileWidgetParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		widgets, info, err := api.ListTurnstileWidgets(ctx, account, params)
		if err != nil {
			return err
		}
		for _, widget := range widgets {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				widget.SiteKey,
				cloudflareResourceName(account.Identifier, widget.Name, widget.SiteKey),
				"cloudflare_turnstile_widget",
				"cloudflare",
				map[string]string{"account_id": account.Identifier, "sitekey": widget.SiteKey},
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
