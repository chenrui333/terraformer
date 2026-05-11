// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
	route53resolvertypes "github.com/aws/aws-sdk-go-v2/service/route53resolver/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	route53ResolverRuleResourceType                         = "aws_route53_resolver_rule"
	route53ResolverRuleAssociationResourceType              = "aws_route53_resolver_rule_association"
	route53ResolverEndpointResourceType                     = "aws_route53_resolver_endpoint"
	route53ResolverQueryLogConfigResourceType               = "aws_route53_resolver_query_log_config"
	route53ResolverQueryLogConfigAssociationResourceType    = "aws_route53_resolver_query_log_config_association"
	route53ResolverFirewallDomainListResourceType           = "aws_route53_resolver_firewall_domain_list"
	route53ResolverFirewallRuleGroupResourceType            = "aws_route53_resolver_firewall_rule_group"
	route53ResolverFirewallRuleResourceType                 = "aws_route53_resolver_firewall_rule"
	route53ResolverFirewallRuleGroupAssociationResourceType = "aws_route53_resolver_firewall_rule_group_association"
	route53ResolverFirewallRuleIDSeparator                  = ":"
)

var route53ResolverAllowEmptyValues = []string{"tags."}

type Route53ResolverGenerator struct {
	AWSService
}

func (g *Route53ResolverGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := route53resolver.NewFromConfig(config)

	loaders := []func(*route53resolver.Client) error{
		g.loadResolverEndpoints,
		g.loadResolverRules,
		g.loadResolverRuleAssociations,
		g.loadResolverQueryLogConfigs,
		g.loadResolverQueryLogConfigAssociations,
		g.loadFirewallDomainLists,
		g.loadFirewallRuleGroups,
		g.loadFirewallRuleGroupAssociations,
	}
	for _, loader := range loaders {
		if err := loader(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *Route53ResolverGenerator) loadResolverEndpoints(svc *route53resolver.Client) error {
	p := route53resolver.NewListResolverEndpointsPaginator(svc, &route53resolver.ListResolverEndpointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpointSummary := range page.ResolverEndpoints {
			endpointID := StringValue(endpointSummary.Id)
			if endpointID == "" {
				continue
			}
			endpoint, err := getRoute53ResolverEndpoint(svc, endpointID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			ipAddresses, err := listRoute53ResolverEndpointIPAddresses(svc, endpointID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newRoute53ResolverEndpointResource(endpoint, ipAddresses); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *Route53ResolverGenerator) loadResolverRules(svc *route53resolver.Client) error {
	p := route53resolver.NewListResolverRulesPaginator(svc, &route53resolver.ListResolverRulesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, ruleSummary := range page.ResolverRules {
			ruleID := StringValue(ruleSummary.Id)
			if ruleID == "" {
				continue
			}
			rule, err := getRoute53ResolverRule(svc, ruleID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newRoute53ResolverRuleResource(rule); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *Route53ResolverGenerator) loadResolverRuleAssociations(svc *route53resolver.Client) error {
	p := route53resolver.NewListResolverRuleAssociationsPaginator(svc, &route53resolver.ListResolverRuleAssociationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, associationSummary := range page.ResolverRuleAssociations {
			associationID := StringValue(associationSummary.Id)
			if associationID == "" {
				continue
			}
			association, err := getRoute53ResolverRuleAssociation(svc, associationID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newRoute53ResolverRuleAssociationResource(association); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *Route53ResolverGenerator) loadResolverQueryLogConfigs(svc *route53resolver.Client) error {
	p := route53resolver.NewListResolverQueryLogConfigsPaginator(svc, &route53resolver.ListResolverQueryLogConfigsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, configSummary := range page.ResolverQueryLogConfigs {
			configID := StringValue(configSummary.Id)
			if configID == "" {
				continue
			}
			config, err := getRoute53ResolverQueryLogConfig(svc, configID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newRoute53ResolverQueryLogConfigResource(config); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *Route53ResolverGenerator) loadResolverQueryLogConfigAssociations(svc *route53resolver.Client) error {
	p := route53resolver.NewListResolverQueryLogConfigAssociationsPaginator(svc, &route53resolver.ListResolverQueryLogConfigAssociationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, associationSummary := range page.ResolverQueryLogConfigAssociations {
			associationID := StringValue(associationSummary.Id)
			if associationID == "" {
				continue
			}
			association, err := getRoute53ResolverQueryLogConfigAssociation(svc, associationID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newRoute53ResolverQueryLogConfigAssociationResource(association); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *Route53ResolverGenerator) loadFirewallDomainLists(svc *route53resolver.Client) error {
	p := route53resolver.NewListFirewallDomainListsPaginator(svc, &route53resolver.ListFirewallDomainListsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, domainListSummary := range page.FirewallDomainLists {
			domainListID := StringValue(domainListSummary.Id)
			if domainListID == "" || StringValue(domainListSummary.ManagedOwnerName) != "" {
				continue
			}
			domainList, err := getRoute53ResolverFirewallDomainList(svc, domainListID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			domains, err := listRoute53ResolverFirewallDomains(svc, domainListID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newRoute53ResolverFirewallDomainListResource(domainList, domains); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *Route53ResolverGenerator) loadFirewallRuleGroups(svc *route53resolver.Client) error {
	p := route53resolver.NewListFirewallRuleGroupsPaginator(svc, &route53resolver.ListFirewallRuleGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, ruleGroupSummary := range page.FirewallRuleGroups {
			ruleGroupID := StringValue(ruleGroupSummary.Id)
			if ruleGroupID == "" || ruleGroupSummary.ShareStatus == route53resolvertypes.ShareStatusSharedWithMe {
				continue
			}
			ruleGroup, err := getRoute53ResolverFirewallRuleGroup(svc, ruleGroupID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newRoute53ResolverFirewallRuleGroupResource(ruleGroup); ok {
				g.Resources = append(g.Resources, resource)
				if err := g.loadFirewallRules(svc, ruleGroupID); err != nil {
					log.Printf("skipping Route 53 Resolver firewall rules for rule group %s: %v", ruleGroupID, err)
				}
			}
		}
	}
	return nil
}

func (g *Route53ResolverGenerator) loadFirewallRules(svc *route53resolver.Client, ruleGroupID string) error {
	if ruleGroupID == "" {
		return nil
	}
	p := route53resolver.NewListFirewallRulesPaginator(svc, &route53resolver.ListFirewallRulesInput{
		FirewallRuleGroupId: aws.String(ruleGroupID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, firewallRule := range page.FirewallRules {
			if resource, ok := newRoute53ResolverFirewallRuleResource(firewallRule); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *Route53ResolverGenerator) loadFirewallRuleGroupAssociations(svc *route53resolver.Client) error {
	p := route53resolver.NewListFirewallRuleGroupAssociationsPaginator(svc, &route53resolver.ListFirewallRuleGroupAssociationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, associationSummary := range page.FirewallRuleGroupAssociations {
			associationID := StringValue(associationSummary.Id)
			if associationID == "" || StringValue(associationSummary.ManagedOwnerName) != "" {
				continue
			}
			association, err := getRoute53ResolverFirewallRuleGroupAssociation(svc, associationID)
			if route53ResolverResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newRoute53ResolverFirewallRuleGroupAssociationResource(association); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newRoute53ResolverEndpointResource(endpoint *route53resolvertypes.ResolverEndpoint, ipAddresses []route53resolvertypes.IpAddressResponse) (terraformutils.Resource, bool) {
	if !route53ResolverEndpointImportable(endpoint, ipAddresses) {
		return terraformutils.Resource{}, false
	}
	endpointID := StringValue(endpoint.Id)
	return terraformutils.NewResource(
		endpointID,
		route53ResolverResourceName("endpoint", StringValue(endpoint.Name), endpointID),
		route53ResolverEndpointResourceType,
		"aws",
		map[string]string{
			"direction": string(endpoint.Direction),
			"name":      StringValue(endpoint.Name),
		},
		route53ResolverAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRoute53ResolverRuleResource(rule *route53resolvertypes.ResolverRule) (terraformutils.Resource, bool) {
	if !route53ResolverRuleImportable(rule) {
		return terraformutils.Resource{}, false
	}
	ruleID := StringValue(rule.Id)
	attributes := map[string]string{
		"domain_name": trimRoute53ResolverTrailingPeriod(StringValue(rule.DomainName)),
		"name":        StringValue(rule.Name),
		"rule_type":   string(rule.RuleType),
	}
	putRoute53ResolverString(attributes, "resolver_endpoint_id", StringValue(rule.ResolverEndpointId))
	return terraformutils.NewResource(
		ruleID,
		route53ResolverResourceName("rule", StringValue(rule.Name), StringValue(rule.DomainName), ruleID),
		route53ResolverRuleResourceType,
		"aws",
		attributes,
		route53ResolverAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRoute53ResolverRuleAssociationResource(association *route53resolvertypes.ResolverRuleAssociation) (terraformutils.Resource, bool) {
	if !route53ResolverRuleAssociationImportable(association) {
		return terraformutils.Resource{}, false
	}
	associationID := StringValue(association.Id)
	return terraformutils.NewResource(
		associationID,
		route53ResolverResourceName("rule_association", StringValue(association.ResolverRuleId), StringValue(association.VPCId), associationID),
		route53ResolverRuleAssociationResourceType,
		"aws",
		map[string]string{
			"name":             StringValue(association.Name),
			"resolver_rule_id": StringValue(association.ResolverRuleId),
			"vpc_id":           StringValue(association.VPCId),
		},
		route53ResolverAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRoute53ResolverQueryLogConfigResource(config *route53resolvertypes.ResolverQueryLogConfig) (terraformutils.Resource, bool) {
	if !route53ResolverQueryLogConfigImportable(config) {
		return terraformutils.Resource{}, false
	}
	configID := StringValue(config.Id)
	return terraformutils.NewResource(
		configID,
		route53ResolverResourceName("query_log_config", StringValue(config.Name), configID),
		route53ResolverQueryLogConfigResourceType,
		"aws",
		map[string]string{
			"destination_arn": StringValue(config.DestinationArn),
			"name":            StringValue(config.Name),
		},
		route53ResolverAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRoute53ResolverQueryLogConfigAssociationResource(association *route53resolvertypes.ResolverQueryLogConfigAssociation) (terraformutils.Resource, bool) {
	if !route53ResolverQueryLogConfigAssociationImportable(association) {
		return terraformutils.Resource{}, false
	}
	associationID := StringValue(association.Id)
	return terraformutils.NewResource(
		associationID,
		route53ResolverResourceName("query_log_config_association", StringValue(association.ResolverQueryLogConfigId), StringValue(association.ResourceId), associationID),
		route53ResolverQueryLogConfigAssociationResourceType,
		"aws",
		map[string]string{
			"resolver_query_log_config_id": StringValue(association.ResolverQueryLogConfigId),
			"resource_id":                  StringValue(association.ResourceId),
		},
		route53ResolverAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRoute53ResolverFirewallDomainListResource(domainList *route53resolvertypes.FirewallDomainList, domains []string) (terraformutils.Resource, bool) {
	if !route53ResolverFirewallDomainListImportable(domainList) {
		return terraformutils.Resource{}, false
	}
	domainListID := StringValue(domainList.Id)
	attributes := map[string]string{
		"name": StringValue(domainList.Name),
	}
	putRoute53ResolverListAttributes(attributes, "domains", domains)
	return terraformutils.NewResource(
		domainListID,
		route53ResolverResourceName("firewall_domain_list", StringValue(domainList.Name), domainListID),
		route53ResolverFirewallDomainListResourceType,
		"aws",
		attributes,
		route53ResolverAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRoute53ResolverFirewallRuleGroupResource(ruleGroup *route53resolvertypes.FirewallRuleGroup) (terraformutils.Resource, bool) {
	if !route53ResolverFirewallRuleGroupImportable(ruleGroup) {
		return terraformutils.Resource{}, false
	}
	ruleGroupID := StringValue(ruleGroup.Id)
	return terraformutils.NewResource(
		ruleGroupID,
		route53ResolverResourceName("firewall_rule_group", StringValue(ruleGroup.Name), ruleGroupID),
		route53ResolverFirewallRuleGroupResourceType,
		"aws",
		map[string]string{
			"name": StringValue(ruleGroup.Name),
		},
		route53ResolverAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRoute53ResolverFirewallRuleResource(rule route53resolvertypes.FirewallRule) (terraformutils.Resource, bool) {
	if !route53ResolverFirewallRuleImportable(rule) {
		return terraformutils.Resource{}, false
	}
	ruleID := route53ResolverFirewallRuleImportID(StringValue(rule.FirewallRuleGroupId), route53ResolverFirewallRuleIdentifier(rule))
	attributes := map[string]string{
		"action":                 string(rule.Action),
		"firewall_rule_group_id": StringValue(rule.FirewallRuleGroupId),
		"name":                   StringValue(rule.Name),
		"priority":               strconv.Itoa(int(aws.ToInt32(rule.Priority))),
	}
	putRoute53ResolverString(attributes, "block_override_dns_type", string(rule.BlockOverrideDnsType))
	putRoute53ResolverString(attributes, "block_override_domain", StringValue(rule.BlockOverrideDomain))
	putRoute53ResolverInt32(attributes, "block_override_ttl", rule.BlockOverrideTtl)
	putRoute53ResolverString(attributes, "block_response", string(rule.BlockResponse))
	putRoute53ResolverString(attributes, "confidence_threshold", string(rule.ConfidenceThreshold))
	putRoute53ResolverString(attributes, "dns_threat_protection", string(rule.DnsThreatProtection))
	putRoute53ResolverString(attributes, "firewall_domain_list_id", StringValue(rule.FirewallDomainListId))
	putRoute53ResolverString(attributes, "firewall_domain_redirection_action", string(rule.FirewallDomainRedirectionAction))
	putRoute53ResolverString(attributes, "firewall_threat_protection_id", StringValue(rule.FirewallThreatProtectionId))
	putRoute53ResolverString(attributes, "q_type", StringValue(rule.Qtype))
	return terraformutils.NewResource(
		ruleID,
		route53ResolverResourceName("firewall_rule", StringValue(rule.FirewallRuleGroupId), StringValue(rule.Name), route53ResolverFirewallRuleIdentifier(rule)),
		route53ResolverFirewallRuleResourceType,
		"aws",
		attributes,
		route53ResolverAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRoute53ResolverFirewallRuleGroupAssociationResource(association *route53resolvertypes.FirewallRuleGroupAssociation) (terraformutils.Resource, bool) {
	if !route53ResolverFirewallRuleGroupAssociationImportable(association) {
		return terraformutils.Resource{}, false
	}
	associationID := StringValue(association.Id)
	return terraformutils.NewResource(
		associationID,
		route53ResolverResourceName("firewall_rule_group_association", StringValue(association.FirewallRuleGroupId), StringValue(association.VpcId), associationID),
		route53ResolverFirewallRuleGroupAssociationResourceType,
		"aws",
		map[string]string{
			"firewall_rule_group_id": StringValue(association.FirewallRuleGroupId),
			"mutation_protection":    string(association.MutationProtection),
			"name":                   StringValue(association.Name),
			"priority":               strconv.Itoa(int(aws.ToInt32(association.Priority))),
			"vpc_id":                 StringValue(association.VpcId),
		},
		route53ResolverAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func route53ResolverEndpointImportable(endpoint *route53resolvertypes.ResolverEndpoint, ipAddresses []route53resolvertypes.IpAddressResponse) bool {
	return endpoint != nil &&
		StringValue(endpoint.Id) != "" &&
		endpoint.Direction != "" &&
		endpoint.Status == route53resolvertypes.ResolverEndpointStatusOperational &&
		len(endpoint.SecurityGroupIds) > 0 &&
		route53ResolverEndpointHasImportableIP(ipAddresses)
}

func route53ResolverEndpointHasImportableIP(ipAddresses []route53resolvertypes.IpAddressResponse) bool {
	for _, ipAddress := range ipAddresses {
		if StringValue(ipAddress.SubnetId) != "" && ipAddress.Status != route53resolvertypes.IpAddressStatusDeleting {
			return true
		}
	}
	return false
}

func route53ResolverRuleImportable(rule *route53resolvertypes.ResolverRule) bool {
	return rule != nil &&
		StringValue(rule.Id) != "" &&
		StringValue(rule.DomainName) != "" &&
		rule.RuleType != "" &&
		rule.Status == route53resolvertypes.ResolverRuleStatusComplete &&
		rule.ShareStatus != route53resolvertypes.ShareStatusSharedWithMe
}

func route53ResolverRuleAssociationImportable(association *route53resolvertypes.ResolverRuleAssociation) bool {
	return association != nil &&
		StringValue(association.Id) != "" &&
		StringValue(association.ResolverRuleId) != "" &&
		StringValue(association.VPCId) != "" &&
		association.Status == route53resolvertypes.ResolverRuleAssociationStatusComplete
}

func route53ResolverQueryLogConfigImportable(config *route53resolvertypes.ResolverQueryLogConfig) bool {
	return config != nil &&
		StringValue(config.Id) != "" &&
		StringValue(config.DestinationArn) != "" &&
		StringValue(config.Name) != "" &&
		config.Status == route53resolvertypes.ResolverQueryLogConfigStatusCreated &&
		config.ShareStatus != route53resolvertypes.ShareStatusSharedWithMe
}

func route53ResolverQueryLogConfigAssociationImportable(association *route53resolvertypes.ResolverQueryLogConfigAssociation) bool {
	return association != nil &&
		StringValue(association.Id) != "" &&
		StringValue(association.ResolverQueryLogConfigId) != "" &&
		StringValue(association.ResourceId) != "" &&
		association.Status == route53resolvertypes.ResolverQueryLogConfigAssociationStatusActive
}

func route53ResolverFirewallDomainListImportable(domainList *route53resolvertypes.FirewallDomainList) bool {
	return domainList != nil &&
		StringValue(domainList.Id) != "" &&
		StringValue(domainList.ManagedOwnerName) == "" &&
		StringValue(domainList.Name) != "" &&
		domainList.Status == route53resolvertypes.FirewallDomainListStatusComplete
}

func route53ResolverFirewallRuleGroupImportable(ruleGroup *route53resolvertypes.FirewallRuleGroup) bool {
	return ruleGroup != nil &&
		StringValue(ruleGroup.Id) != "" &&
		StringValue(ruleGroup.Name) != "" &&
		ruleGroup.Status == route53resolvertypes.FirewallRuleGroupStatusComplete &&
		ruleGroup.ShareStatus != route53resolvertypes.ShareStatusSharedWithMe
}

func route53ResolverFirewallRuleImportable(rule route53resolvertypes.FirewallRule) bool {
	return StringValue(rule.FirewallRuleGroupId) != "" &&
		StringValue(rule.Name) != "" &&
		rule.Action != "" &&
		rule.Priority != nil &&
		route53ResolverFirewallRuleIdentifier(rule) != ""
}

func route53ResolverFirewallRuleGroupAssociationImportable(association *route53resolvertypes.FirewallRuleGroupAssociation) bool {
	return association != nil &&
		StringValue(association.Id) != "" &&
		StringValue(association.FirewallRuleGroupId) != "" &&
		StringValue(association.ManagedOwnerName) == "" &&
		StringValue(association.Name) != "" &&
		association.Priority != nil &&
		StringValue(association.VpcId) != "" &&
		association.Status == route53resolvertypes.FirewallRuleGroupAssociationStatusComplete
}

func route53ResolverFirewallRuleIdentifier(rule route53resolvertypes.FirewallRule) string {
	if id := StringValue(rule.FirewallDomainListId); id != "" {
		return id
	}
	return StringValue(rule.FirewallThreatProtectionId)
}

func route53ResolverFirewallRuleImportID(ruleGroupID, ruleIdentifier string) string {
	return strings.Join([]string{ruleGroupID, ruleIdentifier}, route53ResolverFirewallRuleIDSeparator)
}

func route53ResolverResourceName(parts ...string) string {
	var name strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name.Len() > 0 {
			name.WriteString("_")
		}
		name.WriteString(strconv.Itoa(len(part)))
		name.WriteString("_")
		name.WriteString(part)
	}
	return name.String()
}

func putRoute53ResolverString(attributes map[string]string, key, value string) {
	if value != "" {
		attributes[key] = value
	}
}

func putRoute53ResolverInt32(attributes map[string]string, key string, value *int32) {
	if value != nil {
		attributes[key] = strconv.Itoa(int(aws.ToInt32(value)))
	}
}

func putRoute53ResolverListAttributes(attributes map[string]string, key string, values []string) {
	attributes[key+".#"] = strconv.Itoa(len(values))
	for i, value := range values {
		attributes[key+"."+strconv.Itoa(i)] = value
	}
}

func trimRoute53ResolverTrailingPeriod(s string) string {
	return strings.TrimSuffix(s, ".")
}

func getRoute53ResolverEndpoint(svc *route53resolver.Client, id string) (*route53resolvertypes.ResolverEndpoint, error) {
	output, err := svc.GetResolverEndpoint(context.TODO(), &route53resolver.GetResolverEndpointInput{
		ResolverEndpointId: aws.String(id),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.ResolverEndpoint, nil
}

func getRoute53ResolverRule(svc *route53resolver.Client, id string) (*route53resolvertypes.ResolverRule, error) {
	output, err := svc.GetResolverRule(context.TODO(), &route53resolver.GetResolverRuleInput{
		ResolverRuleId: aws.String(id),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.ResolverRule, nil
}

func getRoute53ResolverRuleAssociation(svc *route53resolver.Client, id string) (*route53resolvertypes.ResolverRuleAssociation, error) {
	output, err := svc.GetResolverRuleAssociation(context.TODO(), &route53resolver.GetResolverRuleAssociationInput{
		ResolverRuleAssociationId: aws.String(id),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.ResolverRuleAssociation, nil
}

func getRoute53ResolverQueryLogConfig(svc *route53resolver.Client, id string) (*route53resolvertypes.ResolverQueryLogConfig, error) {
	output, err := svc.GetResolverQueryLogConfig(context.TODO(), &route53resolver.GetResolverQueryLogConfigInput{
		ResolverQueryLogConfigId: aws.String(id),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.ResolverQueryLogConfig, nil
}

func getRoute53ResolverQueryLogConfigAssociation(svc *route53resolver.Client, id string) (*route53resolvertypes.ResolverQueryLogConfigAssociation, error) {
	output, err := svc.GetResolverQueryLogConfigAssociation(context.TODO(), &route53resolver.GetResolverQueryLogConfigAssociationInput{
		ResolverQueryLogConfigAssociationId: aws.String(id),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.ResolverQueryLogConfigAssociation, nil
}

func getRoute53ResolverFirewallDomainList(svc *route53resolver.Client, id string) (*route53resolvertypes.FirewallDomainList, error) {
	output, err := svc.GetFirewallDomainList(context.TODO(), &route53resolver.GetFirewallDomainListInput{
		FirewallDomainListId: aws.String(id),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.FirewallDomainList, nil
}

func getRoute53ResolverFirewallRuleGroup(svc *route53resolver.Client, id string) (*route53resolvertypes.FirewallRuleGroup, error) {
	output, err := svc.GetFirewallRuleGroup(context.TODO(), &route53resolver.GetFirewallRuleGroupInput{
		FirewallRuleGroupId: aws.String(id),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.FirewallRuleGroup, nil
}

func getRoute53ResolverFirewallRuleGroupAssociation(svc *route53resolver.Client, id string) (*route53resolvertypes.FirewallRuleGroupAssociation, error) {
	output, err := svc.GetFirewallRuleGroupAssociation(context.TODO(), &route53resolver.GetFirewallRuleGroupAssociationInput{
		FirewallRuleGroupAssociationId: aws.String(id),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.FirewallRuleGroupAssociation, nil
}

func listRoute53ResolverEndpointIPAddresses(svc *route53resolver.Client, endpointID string) ([]route53resolvertypes.IpAddressResponse, error) {
	p := route53resolver.NewListResolverEndpointIpAddressesPaginator(svc, &route53resolver.ListResolverEndpointIpAddressesInput{
		ResolverEndpointId: aws.String(endpointID),
	})
	var ipAddresses []route53resolvertypes.IpAddressResponse
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		ipAddresses = append(ipAddresses, page.IpAddresses...)
	}
	return ipAddresses, nil
}

func listRoute53ResolverFirewallDomains(svc *route53resolver.Client, domainListID string) ([]string, error) {
	p := route53resolver.NewListFirewallDomainsPaginator(svc, &route53resolver.ListFirewallDomainsInput{
		FirewallDomainListId: aws.String(domainListID),
	})
	var domains []string
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		domains = append(domains, page.Domains...)
	}
	return domains, nil
}

func route53ResolverResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var resourceNotFound *route53resolvertypes.ResourceNotFoundException
	if errors.As(err, &resourceNotFound) {
		return true
	}
	var unknownResource *route53resolvertypes.UnknownResourceException
	if errors.As(err, &unknownResource) {
		return true
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "ResourceNotFoundException",
		"UnknownResourceException":
		return true
	default:
		return false
	}
}
