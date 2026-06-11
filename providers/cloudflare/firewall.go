// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"fmt"
	"strings"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

type FirewallGenerator struct {
	CloudflareService
}

func (*FirewallGenerator) createZoneLockdownsResources(api *cf.API, zoneID, zoneName string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}

	zonelockdowns, _, err := api.ListZoneLockdowns(context.Background(), cf.ZoneIdentifier(zoneID), cf.LockdownListParams{})
	if err != nil {
		return resources, err
	}
	for _, zonelockdown := range zonelockdowns {
		resource := terraformutils.NewResource(
			zonelockdown.ID,
			fmt.Sprintf("%s_%s", zoneName, zonelockdown.ID),
			"cloudflare_zone_lockdown",
			"cloudflare",
			map[string]string{
				"zone_id": zoneID,
				"zone":    zoneName,
			},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, zoneID+"/"+zonelockdown.ID)
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *FirewallGenerator) createAccountAccessRuleResources(api *cf.API) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	accountID := g.accountID()
	rules, err := api.ListAccountAccessRules(context.Background(), accountID, cf.AccessRule{}, 1)
	if err != nil {
		return resources, err
	}

	totalPages := rules.TotalPages
	for _, rule := range rules.Result {
		resource := terraformutils.NewResource(
			rule.ID,
			cloudflareResourceName("accounts", accountID, rule.ID),
			"cloudflare_access_rule",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, "accounts/"+accountID+"/"+rule.ID)
		resources = append(resources, resource)
	}

	for page := 2; page <= totalPages; page++ {
		rules, err := api.ListAccountAccessRules(context.Background(), accountID, cf.AccessRule{}, page)
		if err != nil {
			return resources, err
		}
		for _, rule := range rules.Result {
			resource := terraformutils.NewResource(
				rule.ID,
				cloudflareResourceName("accounts", accountID, rule.ID),
				"cloudflare_access_rule",
				"cloudflare",
				map[string]string{"account_id": accountID},
				[]string{},
				map[string]interface{}{},
			)
			setCloudflareImportID(&resource, "accounts/"+accountID+"/"+rule.ID)
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

func (*FirewallGenerator) createZoneAccessRuleResources(api *cf.API, zoneID, zoneName string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	rules, err := api.ListZoneAccessRules(context.Background(), zoneID, cf.AccessRule{}, 1)
	if err != nil {
		return resources, err
	}

	totalPages := rules.TotalPages
	for _, r := range rules.Result {
		if strings.Compare(r.Scope.Type, "organization") != 0 {
			resource := terraformutils.NewResource(
				r.ID,
				fmt.Sprintf("%s_%s", zoneName, r.ID),
				"cloudflare_access_rule",
				"cloudflare",
				map[string]string{
					"zone_id": zoneID,
				},
				[]string{},
				map[string]interface{}{},
			)
			setCloudflareImportID(&resource, "zones/"+zoneID+"/"+r.ID)
			resources = append(resources, resource)
		}
	}

	for page := 2; page <= totalPages; page++ {
		rules, err := api.ListZoneAccessRules(context.Background(), zoneID, cf.AccessRule{}, page)
		if err != nil {
			return resources, err
		}
		for _, r := range rules.Result {
			if strings.Compare(r.Scope.Type, "organization") != 0 {
				resource := terraformutils.NewResource(
					r.ID,
					fmt.Sprintf("%s_%s", zoneName, r.ID),
					"cloudflare_access_rule",
					"cloudflare",
					map[string]string{
						"zone_id": zoneID,
					},
					[]string{},
					map[string]interface{}{},
				)
				setCloudflareImportID(&resource, "zones/"+zoneID+"/"+r.ID)
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

func (*FirewallGenerator) createFilterResources(api *cf.API, zoneID, zoneName string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	filters, _, err := api.Filters(context.Background(), cf.ZoneIdentifier(zoneID), cf.FilterListParams{})
	if err != nil {
		return resources, err
	}

	for _, filter := range filters {
		resource := terraformutils.NewResource(
			filter.ID,
			fmt.Sprintf("%s_%s", zoneName, filter.ID),
			"cloudflare_filter",
			"cloudflare",
			map[string]string{
				"zone_id": zoneID,
			},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, zoneID+"/"+filter.ID)
		resources = append(resources, resource)
	}

	return resources, nil
}

func (*FirewallGenerator) createFirewallRuleResources(api *cf.API, zoneID, zoneName string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}

	fwrules, _, err := api.FirewallRules(context.Background(), cf.ZoneIdentifier(zoneID), cf.FirewallRuleListParams{})
	if err != nil {
		return resources, err
	}
	for _, rule := range fwrules {
		resource := terraformutils.NewResource(
			rule.ID,
			fmt.Sprintf("%s_%s", zoneName, rule.ID),
			"cloudflare_firewall_rule",
			"cloudflare",
			map[string]string{
				"zone_id": zoneID,
			},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, zoneID+"/"+rule.ID)
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *FirewallGenerator) createRateLimitResources(api *cf.API, zoneID, _ string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	rateLimits, err := api.ListAllRateLimits(context.Background(), zoneID)
	if err != nil {
		return resources, err
	}
	for _, rateLimit := range rateLimits {
		resource := terraformutils.NewResource(
			rateLimit.ID,
			fmt.Sprintf("%s_%s", zoneID, rateLimit.ID),
			"cloudflare_rate_limit",
			"cloudflare",
			map[string]string{"zone_id": zoneID},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, zoneID+"/"+rateLimit.ID)
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *FirewallGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	if g.accountID() != "" {
		resources, err := g.createAccountAccessRuleResources(api)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	zones, err := api.ListZones(context.Background())
	if err != nil {
		return err
	}

	funcs := []func(*cf.API, string, string) ([]terraformutils.Resource, error){
		g.createFirewallRuleResources,
		g.createFilterResources,
		g.createZoneAccessRuleResources,
		g.createZoneLockdownsResources,
		g.createRateLimitResources,
	}

	for _, zone := range zones {
		for _, f := range funcs {
			// Getting all firewall filters
			tmpRes, err := f(api, zone.ID, zone.Name)
			if err != nil {
				return err
			}
			g.Resources = append(g.Resources, tmpRes...)
		}
	}

	return nil
}

func (g *FirewallGenerator) PostConvertHook() error {
	for i, resourceRecord := range g.Resources {
		// If Zone Name exists, delete ZoneID
		if _, zoneIDExist := resourceRecord.Item["zone_id"]; zoneIDExist {
			delete(g.Resources[i].Item, "zone")
		}

		if resourceRecord.InstanceInfo.Type == "cloudflare_firewall_rule" {
			if resourceRecord.Item["priority"].(string) == "0" {
				delete(g.Resources[i].Item, "priority")
			}
		}

		// Reference to 'cloudflare_filter' resource in 'cloudflare_firewall_rule'
		if resourceRecord.InstanceInfo.Type == "cloudflare_filter" {
			continue
		}
		filterID := resourceRecord.Item["filter_id"]
		for _, filterResource := range g.Resources {
			if filterResource.InstanceInfo.Type != "cloudflare_filter" {
				continue
			}
			if filterID == filterResource.InstanceState.ID {
				g.Resources[i].Item["filter_id"] = "${cloudflare_filter." + filterResource.ResourceName + ".id}"
			}
		}
	}

	return nil
}
