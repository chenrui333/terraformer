// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

type EmailRoutingGenerator struct {
	CloudflareService
}

type emailRoutingAddress struct {
	ID    string `json:"id,omitempty"`
	Tag   string `json:"tag,omitempty"`
	Email string `json:"email,omitempty"`
}

type emailRoutingRule struct {
	ID   string `json:"id,omitempty"`
	Tag  string `json:"tag,omitempty"`
	Name string `json:"name,omitempty"`
}

type emailRoutingDNSResult struct {
	Record []cf.DNSRecord    `json:"record,omitempty"`
	Errors []json.RawMessage `json:"errors,omitempty"`
}

func cloudflareNotFoundError(err error) bool {
	var notFoundErr *cf.NotFoundError
	return errors.As(err, &notFoundErr)
}

func emailRoutingAddressID(address emailRoutingAddress) string {
	if address.ID != "" {
		return address.ID
	}
	return address.Tag
}

func emailRoutingRuleID(rule emailRoutingRule) string {
	if rule.ID != "" {
		return rule.ID
	}
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
	hasDNSSettings, err := emailRoutingDNSSettingsExist(ctx, api, zone.ID)
	if err != nil {
		if cloudflareNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("get email routing DNS settings for zone %q: %w", zone.ID, err)
	}
	if !hasDNSSettings {
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

func emailRoutingDNSSettingsExist(ctx context.Context, api *cf.API, zoneID string) (bool, error) {
	response, err := api.Raw(ctx, http.MethodGet, fmt.Sprintf("/zones/%s/email/routing/dns", zoneID), nil, nil)
	if err != nil {
		return false, err
	}
	var records []cf.DNSRecord
	if err := json.Unmarshal(response.Result, &records); err == nil {
		return len(records) > 0, nil
	}
	var result emailRoutingDNSResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return false, err
	}
	return len(result.Record) > 0 || len(result.Errors) > 0, nil
}

func listEmailRoutingAddresses(ctx context.Context, api *cf.API, accountID string) ([]emailRoutingAddress, error) {
	var addresses []emailRoutingAddress
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/email/routing/addresses?%s", accountID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageAddresses []emailRoutingAddress
		if err := json.Unmarshal(response.Result, &pageAddresses); err != nil {
			return nil, err
		}
		addresses = append(addresses, pageAddresses...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return addresses, nil
}

func listEmailRoutingRules(ctx context.Context, api *cf.API, zoneID string) ([]emailRoutingRule, error) {
	var rules []emailRoutingRule
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/zones/%s/email/routing/rules?%s", zoneID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageRules []emailRoutingRule
		if err := json.Unmarshal(response.Result, &pageRules); err != nil {
			return nil, err
		}
		rules = append(rules, pageRules...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return rules, nil
}

func (g *EmailRoutingGenerator) appendEmailRoutingAddressResources(ctx context.Context, api *cf.API, account *cf.ResourceContainer) error {
	addresses, err := listEmailRoutingAddresses(ctx, api, account.Identifier)
	if err != nil {
		if cloudflareNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("list email routing destination addresses for account %q: %w", account.Identifier, err)
	}
	for _, address := range addresses {
		addressID := emailRoutingAddressID(address)
		if addressID == "" {
			continue
		}
		resource := terraformutils.NewResource(
			addressID,
			cloudflareResourceName(account.Identifier, address.Email, addressID),
			"cloudflare_email_routing_address",
			"cloudflare",
			map[string]string{"account_id": account.Identifier},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, fmt.Sprintf("%s/%s", account.Identifier, addressID))
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *EmailRoutingGenerator) appendEmailRoutingRuleResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	rules, err := listEmailRoutingRules(ctx, api, zone.ID)
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

	accountID := g.accountID()
	if accountID != "" {
		if err := g.appendEmailRoutingAddressResources(ctx, api, cf.AccountIdentifier(accountID)); err != nil {
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
