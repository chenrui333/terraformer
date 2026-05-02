// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type LoadBalancingGenerator struct {
	CloudflareService
}

func (g *LoadBalancingGenerator) appendLoadBalancerResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	params := cf.ListLoadBalancerParams{PaginationOptions: cf.PaginationOptions{Page: 1, PerPage: cloudflarePageSize}}
	for {
		loadBalancers, err := api.ListLoadBalancers(ctx, cf.ZoneIdentifier(zone.ID), params)
		if err != nil {
			return err
		}
		for _, loadBalancer := range loadBalancers {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				loadBalancer.ID,
				cloudflareResourceName(zone.Name, loadBalancer.Name, loadBalancer.ID),
				"cloudflare_load_balancer",
				"cloudflare",
				map[string]string{"zone_id": zone.ID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if len(loadBalancers) < cloudflarePageSize {
			break
		}
		params.Page++
	}
	return nil
}

func (g *LoadBalancingGenerator) appendHealthcheckResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	healthchecks, err := api.Healthchecks(ctx, zone.ID)
	if err != nil {
		return err
	}
	for _, healthcheck := range healthchecks {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			healthcheck.ID,
			cloudflareResourceName(zone.Name, healthcheck.Name, healthcheck.ID),
			"cloudflare_healthcheck",
			"cloudflare",
			map[string]string{"zone_id": zone.ID},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}

func (g *LoadBalancingGenerator) appendLoadBalancerPoolResources(ctx context.Context, api *cf.API, accountID string) error {
	params := cf.ListLoadBalancerPoolParams{PaginationOptions: cf.PaginationOptions{Page: 1, PerPage: cloudflarePageSize}}
	for {
		pools, err := api.ListLoadBalancerPools(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, pool := range pools {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				pool.ID,
				cloudflareResourceName(accountID, pool.Name, pool.ID),
				"cloudflare_load_balancer_pool",
				"cloudflare",
				map[string]string{"account_id": accountID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if len(pools) < cloudflarePageSize {
			break
		}
		params.Page++
	}
	return nil
}

func (g *LoadBalancingGenerator) appendLoadBalancerMonitorResources(ctx context.Context, api *cf.API, accountID string) error {
	params := cf.ListLoadBalancerMonitorParams{PaginationOptions: cf.PaginationOptions{Page: 1, PerPage: cloudflarePageSize}}
	for {
		monitors, err := api.ListLoadBalancerMonitors(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, monitor := range monitors {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				monitor.ID,
				cloudflareResourceName(accountID, monitor.Description, monitor.ID),
				"cloudflare_load_balancer_monitor",
				"cloudflare",
				map[string]string{"account_id": accountID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if len(monitors) < cloudflarePageSize {
			break
		}
		params.Page++
	}
	return nil
}

func (g *LoadBalancingGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	if g.accountID() != "" {
		if err := g.appendLoadBalancerPoolResources(ctx, api, g.accountID()); err != nil {
			return err
		}
		if err := g.appendLoadBalancerMonitorResources(ctx, api, g.accountID()); err != nil {
			return err
		}
	}

	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		if err := g.appendLoadBalancerResources(ctx, api, zone); err != nil {
			return fmt.Errorf("zone %s load balancers: %w", zone.ID, err)
		}
		if err := g.appendHealthcheckResources(ctx, api, zone); err != nil {
			return fmt.Errorf("zone %s healthchecks: %w", zone.ID, err)
		}
	}
	return nil
}
