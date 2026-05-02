// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type MagicWANGenerator struct {
	CloudflareService
}

func (g *MagicWANGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	accountID := account.Identifier

	greTunnels, err := api.ListMagicTransitGRETunnels(ctx, accountID)
	if err != nil {
		return err
	}
	for _, tunnel := range greTunnels {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			tunnel.ID,
			cloudflareResourceName(accountID, tunnel.Name, tunnel.ID),
			"cloudflare_magic_wan_gre_tunnel",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		))
	}

	ipsecTunnels, err := api.ListMagicTransitIPsecTunnels(ctx, accountID)
	if err != nil {
		return err
	}
	for _, tunnel := range ipsecTunnels {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			tunnel.ID,
			cloudflareResourceName(accountID, tunnel.Name, tunnel.ID),
			"cloudflare_magic_wan_ipsec_tunnel",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		))
	}

	staticRoutes, err := api.ListMagicTransitStaticRoutes(ctx, accountID)
	if err != nil {
		return err
	}
	for _, route := range staticRoutes {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			route.ID,
			cloudflareResourceName(accountID, route.Prefix, route.ID),
			"cloudflare_magic_wan_static_route",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
