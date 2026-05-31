// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	cfv7 "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/turnstile"
)

type TurnstileGenerator struct {
	CloudflareService
}

func (g *TurnstileGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeClientV7()
	if err != nil {
		return err
	}
	accountID, err := g.accountIDRequired()
	if err != nil {
		return err
	}
	params := turnstile.WidgetListParams{
		AccountID: cfv7.String(accountID),
		Page:      cfv7.Float(1),
		PerPage:   cfv7.Float(cloudflarePageSize),
	}
	widgets := api.Turnstile.Widgets.ListAutoPaging(ctx, params)
	for widgets.Next() {
		widget := widgets.Current()
		g.Resources = append(g.Resources, terraformutils.NewResource(
			widget.Sitekey,
			cloudflareResourceName(accountID, widget.Name, widget.Sitekey),
			"cloudflare_turnstile_widget",
			"cloudflare",
			map[string]string{"account_id": accountID, "sitekey": widget.Sitekey},
			[]string{},
			map[string]interface{}{},
		))
	}
	return widgets.Err()
}
