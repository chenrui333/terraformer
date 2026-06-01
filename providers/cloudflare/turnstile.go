// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/cloudflare/cloudflare-go/v7/turnstile"
)

type TurnstileGenerator struct {
	CloudflareService
}

func (g *TurnstileGenerator) InitResources() error {
	ctx := context.Background()
	opts, err := g.cloudflareV7Options()
	if err != nil {
		return err
	}
	api := turnstile.NewWidgetService(opts...)
	accountID, err := g.accountIDRequired()
	if err != nil {
		return err
	}
	params := turnstile.WidgetListParams{}
	params.AccountID.Value = accountID
	params.AccountID.Present = true
	params.Page.Value = 1
	params.Page.Present = true
	params.PerPage.Value = cloudflarePageSize
	params.PerPage.Present = true
	widgets := api.ListAutoPaging(ctx, params)
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
