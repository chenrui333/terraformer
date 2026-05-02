// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type LoadBalancingGenerator struct {
	CloudflareService
}

func addStringAttribute(attributes map[string]string, key, value string) {
	if value != "" {
		attributes[key] = value
	}
}

func addStringListAttributes(attributes map[string]string, key string, values []string) {
	if len(values) == 0 {
		return
	}
	attributes[key+".#"] = strconv.Itoa(len(values))
	for index, value := range values {
		attributes[fmt.Sprintf("%s.%d", key, index)] = value
	}
}

func addPrimitiveMapAttributes[T ~float64](attributes map[string]string, key string, values map[string]T) {
	if len(values) == 0 {
		return
	}
	attributes[key+".%"] = strconv.Itoa(len(values))
	for mapKey, value := range values {
		attributes[key+"."+mapKey] = strconv.FormatFloat(float64(value), 'f', -1, 64)
	}
}

func addStringListMapAttributes(attributes map[string]string, key string, values map[string][]string) {
	if len(values) == 0 {
		return
	}
	attributes[key+".%"] = strconv.Itoa(len(values))
	for mapKey, list := range values {
		addStringListAttributes(attributes, key+"."+mapKey, list)
	}
}

func addLoadBalancerRuleFixedResponseAttributes(attributes map[string]string, prefix string, response *cf.LoadBalancerFixedResponseData) {
	if response == nil {
		return
	}
	addStringAttribute(attributes, prefix+".content_type", response.ContentType)
	addStringAttribute(attributes, prefix+".location", response.Location)
	addStringAttribute(attributes, prefix+".message_body", response.MessageBody)
	if response.StatusCode != 0 {
		attributes[prefix+".status_code"] = strconv.Itoa(response.StatusCode)
	}
}

func addLoadBalancerRuleOverrideAttributes(attributes map[string]string, prefix string, overrides cf.LoadBalancerRuleOverrides) {
	if overrides.AdaptiveRouting != nil && overrides.AdaptiveRouting.FailoverAcrossPools != nil {
		attributes[prefix+".adaptive_routing.failover_across_pools"] = strconv.FormatBool(*overrides.AdaptiveRouting.FailoverAcrossPools)
	}
	addStringAttribute(attributes, prefix+".fallback_pool", overrides.FallbackPool)
	if overrides.LocationStrategy != nil {
		addStringAttribute(attributes, prefix+".location_strategy.mode", overrides.LocationStrategy.Mode)
		addStringAttribute(attributes, prefix+".location_strategy.prefer_ecs", overrides.LocationStrategy.PreferECS)
	}
	if overrides.RandomSteering != nil {
		attributes[prefix+".random_steering.default_weight"] = strconv.FormatFloat(overrides.RandomSteering.DefaultWeight, 'f', -1, 64)
		addPrimitiveMapAttributes(attributes, prefix+".random_steering.pool_weights", overrides.RandomSteering.PoolWeights)
	}
	addStringListMapAttributes(attributes, prefix+".country_pools", overrides.CountryPools)
	addStringListAttributes(attributes, prefix+".default_pools", overrides.DefaultPools)
	addStringListMapAttributes(attributes, prefix+".pop_pools", overrides.PoPPools)
	addStringListMapAttributes(attributes, prefix+".region_pools", overrides.RegionPools)
	addStringAttribute(attributes, prefix+".session_affinity", overrides.Persistence)
	if overrides.PersistenceTTL != nil {
		attributes[prefix+".session_affinity_ttl"] = strconv.FormatUint(uint64(*overrides.PersistenceTTL), 10)
	}
	addLoadBalancerRuleSessionAffinityAttributes(attributes, prefix+".session_affinity_attributes", overrides.SessionAffinityAttrs)
	addStringAttribute(attributes, prefix+".steering_policy", overrides.SteeringPolicy)
	if overrides.TTL != 0 {
		attributes[prefix+".ttl"] = strconv.FormatUint(uint64(overrides.TTL), 10)
	}
}

func addLoadBalancerRuleSessionAffinityAttributes(
	attributes map[string]string,
	prefix string,
	sessionAffinity *cf.LoadBalancerRuleOverridesSessionAffinityAttrs,
) {
	if sessionAffinity == nil {
		return
	}
	addStringListAttributes(attributes, prefix+".headers", sessionAffinity.Headers)
	if sessionAffinity.RequireAllHeaders != nil {
		attributes[prefix+".require_all_headers"] = strconv.FormatBool(*sessionAffinity.RequireAllHeaders)
	}
	addStringAttribute(attributes, prefix+".samesite", sessionAffinity.SameSite)
	addStringAttribute(attributes, prefix+".secure", sessionAffinity.Secure)
	addStringAttribute(attributes, prefix+".zero_downtime_failover", sessionAffinity.ZeroDowntimeFailover)
}

func addLoadBalancerRulesAttributes(attributes map[string]string, rules []*cf.LoadBalancerRule) {
	if len(rules) == 0 {
		return
	}
	attributes["rules.#"] = strconv.Itoa(len(rules))
	for index, rule := range rules {
		prefix := fmt.Sprintf("rules.%d", index)
		addStringAttribute(attributes, prefix+".condition", rule.Condition)
		attributes[prefix+".disabled"] = strconv.FormatBool(rule.Disabled)
		addLoadBalancerRuleFixedResponseAttributes(attributes, prefix+".fixed_response", rule.FixedResponse)
		addStringAttribute(attributes, prefix+".name", rule.Name)
		addLoadBalancerRuleOverrideAttributes(attributes, prefix+".overrides", rule.Overrides)
		attributes[prefix+".priority"] = strconv.Itoa(rule.Priority)
		attributes[prefix+".terminates"] = strconv.FormatBool(rule.Terminates)
	}
}

func loadBalancerRuleAdditionalFields(rules []*cf.LoadBalancerRule) []map[string]interface{} {
	fields := make([]map[string]interface{}, 0, len(rules))
	for _, rule := range rules {
		field := map[string]interface{}{
			"condition":  rule.Condition,
			"disabled":   rule.Disabled,
			"name":       rule.Name,
			"priority":   rule.Priority,
			"terminates": rule.Terminates,
		}
		if rule.FixedResponse != nil {
			field["fixed_response"] = loadBalancerFixedResponseAdditionalFields(rule.FixedResponse)
		}
		if overrides := loadBalancerOverrideAdditionalFields(rule.Overrides); len(overrides) > 0 {
			field["overrides"] = overrides
		}
		fields = append(fields, field)
	}
	return fields
}

func loadBalancerFixedResponseAdditionalFields(response *cf.LoadBalancerFixedResponseData) map[string]interface{} {
	fields := map[string]interface{}{}
	if response.ContentType != "" {
		fields["content_type"] = response.ContentType
	}
	if response.Location != "" {
		fields["location"] = response.Location
	}
	if response.MessageBody != "" {
		fields["message_body"] = response.MessageBody
	}
	if response.StatusCode != 0 {
		fields["status_code"] = response.StatusCode
	}
	return fields
}

func loadBalancerOverrideAdditionalFields(overrides cf.LoadBalancerRuleOverrides) map[string]interface{} {
	fields := map[string]interface{}{}
	if overrides.AdaptiveRouting != nil && overrides.AdaptiveRouting.FailoverAcrossPools != nil {
		fields["adaptive_routing"] = map[string]interface{}{"failover_across_pools": *overrides.AdaptiveRouting.FailoverAcrossPools}
	}
	if len(overrides.CountryPools) > 0 {
		fields["country_pools"] = overrides.CountryPools
	}
	if len(overrides.DefaultPools) > 0 {
		fields["default_pools"] = overrides.DefaultPools
	}
	if overrides.FallbackPool != "" {
		fields["fallback_pool"] = overrides.FallbackPool
	}
	if overrides.LocationStrategy != nil {
		fields["location_strategy"] = loadBalancerLocationStrategyAdditionalFields(overrides.LocationStrategy)
	}
	if len(overrides.PoPPools) > 0 {
		fields["pop_pools"] = overrides.PoPPools
	}
	if overrides.RandomSteering != nil {
		fields["random_steering"] = loadBalancerRandomSteeringAdditionalFields(overrides.RandomSteering)
	}
	if len(overrides.RegionPools) > 0 {
		fields["region_pools"] = overrides.RegionPools
	}
	if overrides.Persistence != "" {
		fields["session_affinity"] = overrides.Persistence
	}
	if overrides.SessionAffinityAttrs != nil {
		fields["session_affinity_attributes"] = loadBalancerSessionAffinityAdditionalFields(overrides.SessionAffinityAttrs)
	}
	if overrides.PersistenceTTL != nil {
		fields["session_affinity_ttl"] = *overrides.PersistenceTTL
	}
	if overrides.SteeringPolicy != "" {
		fields["steering_policy"] = overrides.SteeringPolicy
	}
	if overrides.TTL != 0 {
		fields["ttl"] = overrides.TTL
	}
	return fields
}

func loadBalancerLocationStrategyAdditionalFields(strategy *cf.LocationStrategy) map[string]interface{} {
	fields := map[string]interface{}{}
	if strategy.Mode != "" {
		fields["mode"] = strategy.Mode
	}
	if strategy.PreferECS != "" {
		fields["prefer_ecs"] = strategy.PreferECS
	}
	return fields
}

func loadBalancerRandomSteeringAdditionalFields(randomSteering *cf.RandomSteering) map[string]interface{} {
	fields := map[string]interface{}{"default_weight": randomSteering.DefaultWeight}
	if len(randomSteering.PoolWeights) > 0 {
		fields["pool_weights"] = randomSteering.PoolWeights
	}
	return fields
}

func loadBalancerSessionAffinityAdditionalFields(sessionAffinity *cf.LoadBalancerRuleOverridesSessionAffinityAttrs) map[string]interface{} {
	fields := map[string]interface{}{}
	if len(sessionAffinity.Headers) > 0 {
		fields["headers"] = sessionAffinity.Headers
	}
	if sessionAffinity.RequireAllHeaders != nil {
		fields["require_all_headers"] = *sessionAffinity.RequireAllHeaders
	}
	if sessionAffinity.SameSite != "" {
		fields["samesite"] = sessionAffinity.SameSite
	}
	if sessionAffinity.Secure != "" {
		fields["secure"] = sessionAffinity.Secure
	}
	if sessionAffinity.ZeroDowntimeFailover != "" {
		fields["zero_downtime_failover"] = sessionAffinity.ZeroDowntimeFailover
	}
	return fields
}

func (g *LoadBalancingGenerator) appendLoadBalancerResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	params := cf.ListLoadBalancerParams{PaginationOptions: cf.PaginationOptions{Page: 1, PerPage: cloudflarePageSize}}
	for {
		loadBalancers, err := api.ListLoadBalancers(ctx, cf.ZoneIdentifier(zone.ID), params)
		if err != nil {
			return err
		}
		for _, loadBalancer := range loadBalancers {
			attributes := map[string]string{"zone_id": zone.ID}
			addLoadBalancerRulesAttributes(attributes, loadBalancer.Rules)
			additionalFields := map[string]interface{}{}
			if len(loadBalancer.Rules) > 0 {
				additionalFields["rules"] = loadBalancerRuleAdditionalFields(loadBalancer.Rules)
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				loadBalancer.ID,
				cloudflareResourceName(zone.Name, loadBalancer.Name, loadBalancer.ID),
				"cloudflare_load_balancer",
				"cloudflare",
				attributes,
				[]string{},
				additionalFields,
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
