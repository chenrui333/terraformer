// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type MagicWANGenerator struct {
	CloudflareService
}

func magicWANGRETunnelAttributes(accountID string, tunnel cf.MagicTransitGRETunnel) (map[string]string, bool) {
	if tunnel.Name == "" || tunnel.CloudflareGREEndpoint == "" ||
		tunnel.CustomerGREEndpoint == "" || tunnel.InterfaceAddress == "" {
		return nil, false
	}
	attributes := map[string]string{
		"account_id":              accountID,
		"cloudflare_gre_endpoint": tunnel.CloudflareGREEndpoint,
		"customer_gre_endpoint":   tunnel.CustomerGREEndpoint,
		"interface_address":       tunnel.InterfaceAddress,
		"name":                    tunnel.Name,
		"mtu":                     strconv.FormatUint(uint64(tunnel.MTU), 10),
		"ttl":                     strconv.FormatUint(uint64(tunnel.TTL), 10),
	}
	addStringAttribute(attributes, "description", tunnel.Description)
	if tunnel.HealthCheck != nil {
		attributes["health_check.enabled"] = strconv.FormatBool(tunnel.HealthCheck.Enabled)
		addStringAttribute(attributes, "health_check.target.saved", tunnel.HealthCheck.Target)
		addStringAttribute(attributes, "health_check.type", tunnel.HealthCheck.Type)
	}
	return attributes, true
}

func magicWANIPsecTunnelAttributes(accountID string, tunnel cf.MagicTransitIPsecTunnel) (map[string]string, bool) {
	if tunnel.Name == "" || tunnel.CloudflareEndpoint == "" || tunnel.InterfaceAddress == "" {
		return nil, false
	}
	attributes := map[string]string{
		"account_id":          accountID,
		"cloudflare_endpoint": tunnel.CloudflareEndpoint,
		"interface_address":   tunnel.InterfaceAddress,
		"name":                tunnel.Name,
		"allow_null_cipher":   strconv.FormatBool(tunnel.AllowNullCipher),
	}
	addStringAttribute(attributes, "customer_endpoint", tunnel.CustomerEndpoint)
	addStringAttribute(attributes, "description", tunnel.Description)
	addStringAttribute(attributes, "psk", tunnel.Psk)
	if tunnel.ReplayProtection != nil {
		attributes["replay_protection"] = strconv.FormatBool(*tunnel.ReplayProtection)
	} else {
		attributes["replay_protection"] = "false"
	}
	if tunnel.RemoteIdentities != nil {
		addStringAttribute(attributes, "custom_remote_identities.fqdn_id", tunnel.RemoteIdentities.FQDNID)
	}
	if tunnel.HealthCheck != nil {
		attributes["health_check.enabled"] = strconv.FormatBool(tunnel.HealthCheck.Enabled)
		addStringAttribute(attributes, "health_check.direction", tunnel.HealthCheck.Direction)
		addStringAttribute(attributes, "health_check.rate", tunnel.HealthCheck.Rate)
		addStringAttribute(attributes, "health_check.target.saved", tunnel.HealthCheck.Target)
		addStringAttribute(attributes, "health_check.type", tunnel.HealthCheck.Type)
	}
	return attributes, true
}

func magicWANStaticRouteAttributes(accountID string, route cf.MagicTransitStaticRoute) (map[string]string, bool) {
	if route.Nexthop == "" || route.Prefix == "" {
		return nil, false
	}
	attributes := map[string]string{
		"account_id": accountID,
		"nexthop":    route.Nexthop,
		"prefix":     route.Prefix,
		"priority":   strconv.Itoa(route.Priority),
	}
	addStringAttribute(attributes, "description", route.Description)
	if route.Weight != 0 {
		attributes["weight"] = strconv.Itoa(route.Weight)
	}
	addStringListAttributes(attributes, "scope.colo_names", route.Scope.ColoNames)
	addStringListAttributes(attributes, "scope.colo_regions", route.Scope.ColoRegions)
	return attributes, true
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
		attributes, ok := magicWANGRETunnelAttributes(accountID, tunnel)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			tunnel.ID,
			cloudflareResourceName(accountID, tunnel.Name, tunnel.ID),
			"cloudflare_magic_wan_gre_tunnel",
			"cloudflare",
			attributes,
			[]string{},
			map[string]interface{}{},
		))
	}

	ipsecTunnels, err := api.ListMagicTransitIPsecTunnels(ctx, accountID)
	if err != nil {
		return err
	}
	for _, tunnel := range ipsecTunnels {
		attributes, ok := magicWANIPsecTunnelAttributes(accountID, tunnel)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			tunnel.ID,
			cloudflareResourceName(accountID, tunnel.Name, tunnel.ID),
			"cloudflare_magic_wan_ipsec_tunnel",
			"cloudflare",
			attributes,
			[]string{},
			map[string]interface{}{},
		))
	}

	staticRoutes, err := api.ListMagicTransitStaticRoutes(ctx, accountID)
	if err != nil {
		return err
	}
	for _, route := range staticRoutes {
		attributes, ok := magicWANStaticRouteAttributes(accountID, route)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			route.ID,
			cloudflareResourceName(accountID, route.Prefix, route.ID),
			"cloudflare_magic_wan_static_route",
			"cloudflare",
			attributes,
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
