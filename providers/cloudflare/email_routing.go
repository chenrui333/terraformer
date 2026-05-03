// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"errors"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type EmailRoutingGenerator struct {
	CloudflareService
}

func cloudflareNotFoundError(err error) bool {
	var notFoundErr *cf.NotFoundError
	return errors.As(err, &notFoundErr)
}

func emailRoutingRuleID(rule cf.EmailRoutingRule) string {
	return rule.Tag
}

func emailRoutingCatchAllID(rule cf.EmailRoutingCatchAllRule, zoneID string) string {
	if rule.Tag != "" {
		return rule.Tag
	}
	return zoneID
}

func (g *EmailRoutingGenerator) appendEmailRoutingSettingsResource(zone cf.Zone, settings cf.EmailRoutingSettings) {
	resource := terraformutils.NewResource(
		zone.ID,
		cloudflareResourceName(zone.Name, settings.Tag),
		"cloudflare_email_routing_settings",
		"cloudflare",
		map[string]string{"zone_id": zone.ID},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, zone.ID)
	g.Resources = append(g.Resources, resource)
}

func (g *EmailRoutingGenerator) appendEmailRoutingDNSResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	settings, err := api.GetEmailRoutingDNSSettings(ctx, cf.ZoneIdentifier(zone.ID))
	if err != nil {
		if cloudflareNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("get email routing DNS settings for zone %q: %w", zone.ID, err)
	}
	if len(settings) == 0 {
		return nil
	}

	resource := terraformutils.NewResource(
		zone.ID,
		cloudflareResourceName(zone.Name, "dns"),
		"cloudflare_email_routing_dns",
		"cloudflare",
		map[string]string{"zone_id": zone.ID},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, zone.ID)
	g.Resources = append(g.Resources, resource)
	return nil
}

func (g *EmailRoutingGenerator) appendEmailRoutingAddressResources(ctx context.Context, api *cf.API, account *cf.ResourceContainer) error {
	params := cf.ListEmailRoutingAddressParameters{
		ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize},
	}
	for {
		addresses, info, err := api.ListEmailRoutingDestinationAddresses(ctx, account, params)
		if err != nil {
			if cloudflareNotFoundError(err) {
				return nil
			}
			return fmt.Errorf("list email routing destination addresses for account %q: %w", account.Identifier, err)
		}
		for _, address := range addresses {
			if address.Tag == "" {
				continue
			}
			resource := terraformutils.NewResource(
				address.Tag,
				cloudflareResourceName(account.Identifier, address.Email, address.Tag),
				"cloudflare_email_routing_address",
				"cloudflare",
				map[string]string{"account_id": account.Identifier},
				[]string{},
				map[string]interface{}{},
			)
			setCloudflareImportID(&resource, fmt.Sprintf("%s/%s", account.Identifier, address.Tag))
			g.Resources = append(g.Resources, resource)
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *EmailRoutingGenerator) appendEmailRoutingRuleResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	params := cf.ListEmailRoutingRulesParameters{
		ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize},
	}
	for {
		rules, info, err := api.ListEmailRoutingRules(ctx, cf.ZoneIdentifier(zone.ID), params)
		if err != nil {
			if cloudflareNotFoundError(err) {
				return nil
			}
			return fmt.Errorf("list email routing rules for zone %q: %w", zone.ID, err)
		}
		for _, rule := range rules {
			ruleID := emailRoutingRuleID(rule)
			if ruleID == "" {
				continue
			}
			resource := terraformutils.NewResource(
				ruleID,
				cloudflareResourceName(zone.Name, rule.Name, ruleID),
				"cloudflare_email_routing_rule",
				"cloudflare",
				map[string]string{"zone_id": zone.ID},
				[]string{},
				map[string]interface{}{},
			)
			setCloudflareImportID(&resource, fmt.Sprintf("%s/%s", zone.ID, ruleID))
			g.Resources = append(g.Resources, resource)
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *EmailRoutingGenerator) appendEmailRoutingCatchAllResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	rule, err := api.GetEmailRoutingCatchAllRule(ctx, cf.ZoneIdentifier(zone.ID))
	if err != nil {
		if cloudflareNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("get email routing catch-all rule for zone %q: %w", zone.ID, err)
	}
	ruleID := emailRoutingCatchAllID(rule, zone.ID)

	resource := terraformutils.NewResource(
		ruleID,
		cloudflareResourceName(zone.Name, "catch_all", ruleID),
		"cloudflare_email_routing_catch_all",
		"cloudflare",
		map[string]string{"zone_id": zone.ID},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, zone.ID)
	g.Resources = append(g.Resources, resource)
	return nil
}

func (g *EmailRoutingGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	if account, err := g.accountResourceContainer(); err == nil {
		if err := g.appendEmailRoutingAddressResources(ctx, api, account); err != nil {
			return err
		}
	}

	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		settings, err := api.GetEmailRoutingSettings(ctx, cf.ZoneIdentifier(zone.ID))
		if err != nil {
			if cloudflareNotFoundError(err) {
				continue
			}
			return fmt.Errorf("get email routing settings for zone %q: %w", zone.ID, err)
		}
		if !settings.Enabled {
			continue
		}
		g.appendEmailRoutingSettingsResource(zone, settings)
		if err := g.appendEmailRoutingDNSResource(ctx, api, zone); err != nil {
			return err
		}
		if err := g.appendEmailRoutingRuleResources(ctx, api, zone); err != nil {
			return err
		}
		if err := g.appendEmailRoutingCatchAllResource(ctx, api, zone); err != nil {
			return err
		}
	}
	return nil
}
