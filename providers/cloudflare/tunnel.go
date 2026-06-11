// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

type TunnelGenerator struct {
	CloudflareService
}

type cloudflareTunnelRoute struct {
	ID               string  `json:"id"`
	Network          string  `json:"network"`
	TunnelID         string  `json:"tunnel_id"`
	Comment          string  `json:"comment"`
	DeletedAt        *string `json:"deleted_at"`
	TunType          string  `json:"tun_type"`
	VirtualNetworkID string  `json:"virtual_network_id"`
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
			attributes := map[string]string{
				"account_id": accountID,
				"config_src": tunnelConfigSource(tunnel.RemoteConfig),
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				tunnel.ID,
				cloudflareResourceName(accountID, tunnel.Name, tunnel.ID),
				"cloudflare_zero_trust_tunnel_cloudflared",
				"cloudflare",
				attributes,
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

func (g *TunnelGenerator) appendTunnelRouteResources(ctx context.Context, api *cf.API, accountID string) error {
	routes, err := listTunnelRoutes(ctx, api, accountID)
	if err != nil {
		return err
	}
	for _, route := range routes {
		resource, ok := cloudflareTunnelRouteResource(accountID, route)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func listTunnelRoutes(ctx context.Context, api *cf.API, accountID string) ([]cloudflareTunnelRoute, error) {
	var routes []cloudflareTunnelRoute
	page, cursor := 1, ""
	for {
		response, err := api.Raw(ctx, http.MethodGet, cloudflareTunnelRoutePath(accountID, page, cursor), nil, nil)
		if err != nil {
			return nil, err
		}
		if len(response.Result) == 0 || string(response.Result) == "null" {
			return routes, nil
		}

		var pageRoutes []cloudflareTunnelRoute
		if err := json.Unmarshal(response.Result, &pageRoutes); err != nil {
			return nil, err
		}
		routes = append(routes, pageRoutes...)
		if !cloudflareAdvancePaginationWithItemCount(response.ResultInfo, &page, &cursor, len(pageRoutes)) {
			break
		}
	}
	return routes, nil
}

func cloudflareTunnelRoutePath(accountID string, page int, cursor string) string {
	values := url.Values{}
	values.Set("is_deleted", "false")
	values.Set("per_page", strconv.Itoa(cloudflarePageSize))
	values.Add("tun_types", "cfd_tunnel")
	if cursor != "" {
		values.Set("cursor", cursor)
	} else {
		values.Set("page", strconv.Itoa(page))
	}
	return fmt.Sprintf("/accounts/%s/teamnet/routes?%s", accountID, values.Encode())
}

func cloudflareTunnelRouteResource(accountID string, route cloudflareTunnelRoute) (terraformutils.Resource, bool) {
	if accountID == "" || !cloudflareTunnelRouteImportable(route) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"account_id": accountID,
		"network":    route.Network,
		"tunnel_id":  route.TunnelID,
	}
	if route.Comment != "" {
		attributes["comment"] = route.Comment
	}
	if route.VirtualNetworkID != "" {
		attributes["virtual_network_id"] = route.VirtualNetworkID
	}

	resource := terraformutils.NewResource(
		route.ID,
		cloudflareResourceName(accountID, "tunnel_route", route.Network, route.ID),
		"cloudflare_zero_trust_tunnel_cloudflared_route",
		"cloudflare",
		attributes,
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, accountID+"/"+route.ID)
	return resource, true
}

func cloudflareTunnelRouteImportable(route cloudflareTunnelRoute) bool {
	if route.ID == "" || route.Network == "" || route.TunnelID == "" {
		return false
	}
	if route.DeletedAt != nil && *route.DeletedAt != "" {
		return false
	}
	return route.TunType == "" || route.TunType == "cfd_tunnel"
}

func tunnelConfigSource(remoteConfig bool) string {
	if remoteConfig {
		return "cloudflare"
	}
	return "local"
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
	if err := g.appendTunnelVirtualNetworkResources(ctx, api, account.Identifier); err != nil {
		return err
	}
	return g.appendTunnelRouteResources(ctx, api, account.Identifier)
}
