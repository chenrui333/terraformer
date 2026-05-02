// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type TunnelGenerator struct {
	CloudflareService
}

func (g *TunnelGenerator) appendTunnelResources(ctx context.Context, api *cf.API, accountID string) error {
	isDeleted := false
	params := cf.TunnelListParams{IsDeleted: &isDeleted, ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		tunnels, info, err := api.ListTunnels(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, tunnel := range tunnels {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				tunnel.ID,
				cloudflareResourceName(accountID, tunnel.Name, tunnel.ID),
				"cloudflare_zero_trust_tunnel_cloudflared",
				"cloudflare",
				map[string]string{"account_id": accountID},
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

func (g *TunnelGenerator) appendTunnelVirtualNetworkResources(ctx context.Context, api *cf.API, accountID string) error {
	isDeleted := false
	params := cf.TunnelVirtualNetworksListParams{
		IsDeleted:         &isDeleted,
		PaginationOptions: cf.PaginationOptions{Page: 1, PerPage: cloudflarePageSize},
	}
	for {
		virtualNetworks, err := api.ListTunnelVirtualNetworks(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, virtualNetwork := range virtualNetworks {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				virtualNetwork.ID,
				cloudflareResourceName(accountID, virtualNetwork.Name, virtualNetwork.ID),
				"cloudflare_zero_trust_tunnel_cloudflared_virtual_network",
				"cloudflare",
				map[string]string{"account_id": accountID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if len(virtualNetworks) < cloudflarePageSize {
			break
		}
		params.Page++
	}
	return nil
}

func (g *TunnelGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	if err := g.appendTunnelResources(ctx, api, account.Identifier); err != nil {
		return err
	}
	return g.appendTunnelVirtualNetworkResources(ctx, api, account.Identifier)
}
