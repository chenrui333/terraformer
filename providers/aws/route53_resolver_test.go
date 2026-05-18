// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
	route53resolvertypes "github.com/aws/aws-sdk-go-v2/service/route53resolver/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	testRoute53ResolverEndpointID          = "rslvr-in-1234567890abcdef0"
	testRoute53ResolverRuleID              = "rslvr-rr-1234567890abcdef0"
	testRoute53ResolverRuleAssociationID   = "rslvr-rrassoc-1234567890abcdef0"
	testRoute53ResolverQueryLogConfigID    = "rqlc-1234567890abcdef0"
	testRoute53ResolverQueryLogAssocID     = "rqlca-1234567890abcdef0"
	testRoute53ResolverFirewallDomainID    = "rslvr-fdl-1234567890abcdef0"
	testRoute53ResolverFirewallRuleGroupID = "rslvr-frg-1234567890abcdef0"
	testRoute53ResolverFirewallAssocID     = "rslvr-frgassoc-1234567890abcdef0"
	testRoute53ResolverConfigID            = "rslvr-rc-1234567890abcdef0"
	testRoute53ResolverDNSSECConfigID      = "rslvr-dnssec-1234567890abcdef0"
	testRoute53ResolverFirewallConfigID    = "rslvr-fc-1234567890abcdef0"
	testRoute53ResolverVPCID               = "vpc-1234567890abcdef0"
)

func TestRoute53ResolverResourceIDs(t *testing.T) {
	if got, want := route53ResolverFirewallRuleImportID(testRoute53ResolverFirewallRuleGroupID, testRoute53ResolverFirewallDomainID), testRoute53ResolverFirewallRuleGroupID+":"+testRoute53ResolverFirewallDomainID; got != want {
		t.Fatalf("firewall rule import ID = %q, want %q", got, want)
	}
}

func TestNewRoute53ResolverConfigResources(t *testing.T) {
	resolverConfig, ok := newRoute53ResolverConfigResource(&route53resolvertypes.ResolverConfig{
		AutodefinedReverse: route53resolvertypes.ResolverAutodefinedReverseStatusDisabled,
		Id:                 aws.String(testRoute53ResolverConfigID),
		ResourceId:         aws.String(testRoute53ResolverVPCID),
	})
	assertRoute53ResolverResourceAttributes(t, resolverConfig, ok, route53ResolverConfigResourceType, testRoute53ResolverConfigID,
		[]string{"config", testRoute53ResolverVPCID, testRoute53ResolverConfigID},
		map[string]string{"autodefined_reverse_flag": "DISABLE", "resource_id": testRoute53ResolverVPCID})

	dnssecConfig, ok := newRoute53ResolverDNSSECConfigResource(&route53resolvertypes.ResolverDnssecConfig{
		Id:               aws.String(testRoute53ResolverDNSSECConfigID),
		ResourceId:       aws.String(testRoute53ResolverVPCID),
		ValidationStatus: route53resolvertypes.ResolverDNSSECValidationStatusEnabled,
	})
	assertRoute53ResolverResourceAttributes(t, dnssecConfig, ok, route53ResolverDNSSECConfigResourceType, testRoute53ResolverDNSSECConfigID,
		[]string{"dnssec_config", testRoute53ResolverVPCID, testRoute53ResolverDNSSECConfigID},
		map[string]string{"resource_id": testRoute53ResolverVPCID})

	firewallConfig, ok := newRoute53ResolverFirewallConfigResource(&route53resolvertypes.FirewallConfig{
		FirewallFailOpen: route53resolvertypes.FirewallFailOpenStatusEnabled,
		Id:               aws.String(testRoute53ResolverFirewallConfigID),
		ResourceId:       aws.String(testRoute53ResolverVPCID),
	})
	assertRoute53ResolverResourceAttributes(t, firewallConfig, ok, route53ResolverFirewallConfigResourceType, testRoute53ResolverFirewallConfigID,
		[]string{"firewall_config", testRoute53ResolverVPCID, testRoute53ResolverFirewallConfigID},
		map[string]string{"firewall_fail_open": "ENABLED", "resource_id": testRoute53ResolverVPCID})

	if _, ok := newRoute53ResolverConfigResource(&route53resolvertypes.ResolverConfig{AutodefinedReverse: route53resolvertypes.ResolverAutodefinedReverseStatusEnabled, Id: aws.String(testRoute53ResolverConfigID), ResourceId: aws.String(testRoute53ResolverVPCID)}); ok {
		t.Fatal("default enabled resolver config should be skipped")
	}
	if _, ok := newRoute53ResolverConfigResource(&route53resolvertypes.ResolverConfig{AutodefinedReverse: route53resolvertypes.ResolverAutodefinedReverseStatusEnabling, Id: aws.String(testRoute53ResolverConfigID), ResourceId: aws.String(testRoute53ResolverVPCID)}); ok {
		t.Fatal("pending resolver config should be skipped")
	}
	if _, ok := newRoute53ResolverDNSSECConfigResource(&route53resolvertypes.ResolverDnssecConfig{Id: aws.String(testRoute53ResolverDNSSECConfigID), ResourceId: aws.String(testRoute53ResolverVPCID), ValidationStatus: route53resolvertypes.ResolverDNSSECValidationStatusDisabled}); ok {
		t.Fatal("disabled DNSSEC config should be skipped")
	}
	if _, ok := newRoute53ResolverFirewallConfigResource(&route53resolvertypes.FirewallConfig{FirewallFailOpen: route53resolvertypes.FirewallFailOpenStatusDisabled, Id: aws.String(testRoute53ResolverFirewallConfigID), ResourceId: aws.String(testRoute53ResolverVPCID)}); ok {
		t.Fatal("default disabled firewall config should be skipped")
	}
}

func TestRoute53ResolverConfigPagination(t *testing.T) {
	client := &fakeRoute53ResolverConfigListClient{
		resolverConfigPages: []*route53resolver.ListResolverConfigsOutput{
			{
				NextToken: aws.String("next-resolver"),
				ResolverConfigs: []route53resolvertypes.ResolverConfig{{
					AutodefinedReverse: route53resolvertypes.ResolverAutodefinedReverseStatusDisabled,
					Id:                 aws.String(testRoute53ResolverConfigID),
					ResourceId:         aws.String(testRoute53ResolverVPCID),
				}},
			},
			{ResolverConfigs: []route53resolvertypes.ResolverConfig{{Id: aws.String("rslvr-rc-2"), ResourceId: aws.String("vpc-2")}}},
		},
		dnssecConfigPages: []*route53resolver.ListResolverDnssecConfigsOutput{
			{
				NextToken: aws.String("next-dnssec"),
				ResolverDnssecConfigs: []route53resolvertypes.ResolverDnssecConfig{{
					Id:               aws.String(testRoute53ResolverDNSSECConfigID),
					ResourceId:       aws.String(testRoute53ResolverVPCID),
					ValidationStatus: route53resolvertypes.ResolverDNSSECValidationStatusEnabled,
				}},
			},
			{ResolverDnssecConfigs: []route53resolvertypes.ResolverDnssecConfig{{Id: aws.String("rslvr-dnssec-2"), ResourceId: aws.String("vpc-2")}}},
		},
		firewallConfigPages: []*route53resolver.ListFirewallConfigsOutput{
			{
				FirewallConfigs: []route53resolvertypes.FirewallConfig{{
					FirewallFailOpen: route53resolvertypes.FirewallFailOpenStatusEnabled,
					Id:               aws.String(testRoute53ResolverFirewallConfigID),
					ResourceId:       aws.String(testRoute53ResolverVPCID),
				}},
				NextToken: aws.String("next-firewall"),
			},
			{FirewallConfigs: []route53resolvertypes.FirewallConfig{{Id: aws.String("rslvr-fc-2"), ResourceId: aws.String("vpc-2")}}},
		},
	}

	resolverConfigs, err := listRoute53ResolverConfigs(client)
	if err != nil {
		t.Fatalf("list resolver configs: %v", err)
	}
	if got, want := len(resolverConfigs), 2; got != want {
		t.Fatalf("resolver config count = %d, want %d", got, want)
	}
	if got, want := StringValue(client.resolverConfigInputs[1].NextToken), "next-resolver"; got != want {
		t.Fatalf("resolver config page token = %q, want %q", got, want)
	}

	dnssecConfigs, err := listRoute53ResolverDNSSECConfigs(client)
	if err != nil {
		t.Fatalf("list DNSSEC configs: %v", err)
	}
	if got, want := len(dnssecConfigs), 2; got != want {
		t.Fatalf("DNSSEC config count = %d, want %d", got, want)
	}
	if got, want := StringValue(client.dnssecConfigInputs[1].NextToken), "next-dnssec"; got != want {
		t.Fatalf("DNSSEC config page token = %q, want %q", got, want)
	}

	firewallConfigs, err := listRoute53ResolverFirewallConfigs(client)
	if err != nil {
		t.Fatalf("list firewall configs: %v", err)
	}
	if got, want := len(firewallConfigs), 2; got != want {
		t.Fatalf("firewall config count = %d, want %d", got, want)
	}
	if got, want := StringValue(client.firewallConfigInputs[1].NextToken), "next-firewall"; got != want {
		t.Fatalf("firewall config page token = %q, want %q", got, want)
	}
}

func TestNewRoute53ResolverEndpointResource(t *testing.T) {
	resource, ok := newRoute53ResolverEndpointResource(&route53resolvertypes.ResolverEndpoint{
		Direction:        route53resolvertypes.ResolverEndpointDirectionInbound,
		Id:               aws.String(testRoute53ResolverEndpointID),
		Name:             aws.String("inbound"),
		SecurityGroupIds: []string{"sg-1234567890abcdef0"},
		Status:           route53resolvertypes.ResolverEndpointStatusOperational,
	}, []route53resolvertypes.IpAddressResponse{{Status: route53resolvertypes.IpAddressStatusAttached, SubnetId: aws.String("subnet-1234567890abcdef0")}})
	assertRoute53ResolverResourceAttributes(t, resource, ok, route53ResolverEndpointResourceType, testRoute53ResolverEndpointID,
		[]string{"endpoint", "inbound", testRoute53ResolverEndpointID},
		map[string]string{"direction": "INBOUND", "name": "inbound"})

	if _, ok := newRoute53ResolverEndpointResource(&route53resolvertypes.ResolverEndpoint{Direction: route53resolvertypes.ResolverEndpointDirectionInbound, Id: aws.String(testRoute53ResolverEndpointID), SecurityGroupIds: []string{"sg-1"}, Status: route53resolvertypes.ResolverEndpointStatusDeleting}, []route53resolvertypes.IpAddressResponse{{SubnetId: aws.String("subnet-1")}}); ok {
		t.Fatal("deleting endpoint should be skipped")
	}
	if _, ok := newRoute53ResolverEndpointResource(&route53resolvertypes.ResolverEndpoint{Direction: route53resolvertypes.ResolverEndpointDirectionInbound, Id: aws.String(testRoute53ResolverEndpointID), SecurityGroupIds: []string{"sg-1"}, Status: route53resolvertypes.ResolverEndpointStatusOperational}, nil); ok {
		t.Fatal("endpoint without IP addresses should be skipped")
	}
}

func TestNewRoute53ResolverRuleResource(t *testing.T) {
	resource, ok := newRoute53ResolverRuleResource(&route53resolvertypes.ResolverRule{
		DomainName:         aws.String("example.com."),
		Id:                 aws.String(testRoute53ResolverRuleID),
		Name:               aws.String("example-forward"),
		ResolverEndpointId: aws.String(testRoute53ResolverEndpointID),
		RuleType:           route53resolvertypes.RuleTypeOptionForward,
		ShareStatus:        route53resolvertypes.ShareStatusNotShared,
		Status:             route53resolvertypes.ResolverRuleStatusComplete,
	})
	assertRoute53ResolverResourceAttributes(t, resource, ok, route53ResolverRuleResourceType, testRoute53ResolverRuleID,
		[]string{"rule", "example-forward", "example.com.", testRoute53ResolverRuleID},
		map[string]string{
			"domain_name":          "example.com",
			"name":                 "example-forward",
			"resolver_endpoint_id": testRoute53ResolverEndpointID,
			"rule_type":            "FORWARD",
		})

	if _, ok := newRoute53ResolverRuleResource(&route53resolvertypes.ResolverRule{DomainName: aws.String("example.com."), Id: aws.String(testRoute53ResolverRuleID), RuleType: route53resolvertypes.RuleTypeOptionForward, ShareStatus: route53resolvertypes.ShareStatusSharedWithMe, Status: route53resolvertypes.ResolverRuleStatusComplete}); ok {
		t.Fatal("shared-with-me resolver rule should be skipped")
	}
	if _, ok := newRoute53ResolverRuleResource(&route53resolvertypes.ResolverRule{DomainName: aws.String("."), Id: aws.String("rslvr-autodefined-rr-internet-resolver"), OwnerId: aws.String("Route 53 Resolver"), RuleType: route53resolvertypes.RuleTypeOptionRecursive, ShareStatus: route53resolvertypes.ShareStatusNotShared, Status: route53resolvertypes.ResolverRuleStatusComplete}); ok {
		t.Fatal("AWS-owned autodefined resolver rule should be skipped")
	}
	if _, ok := newRoute53ResolverRuleResource(&route53resolvertypes.ResolverRule{DomainName: aws.String("example.org."), Id: aws.String(testRoute53ResolverRuleID), OwnerId: aws.String("Route 53 Resolver"), RuleType: route53resolvertypes.RuleTypeOptionForward, ShareStatus: route53resolvertypes.ShareStatusNotShared, Status: route53resolvertypes.ResolverRuleStatusComplete}); ok {
		t.Fatal("AWS-owned resolver rule should be skipped")
	}
}

func TestNewRoute53ResolverRuleAssociationResource(t *testing.T) {
	resource, ok := newRoute53ResolverRuleAssociationResource(&route53resolvertypes.ResolverRuleAssociation{
		Id:             aws.String(testRoute53ResolverRuleAssociationID),
		Name:           aws.String("example-association"),
		ResolverRuleId: aws.String(testRoute53ResolverRuleID),
		Status:         route53resolvertypes.ResolverRuleAssociationStatusComplete,
		VPCId:          aws.String(testRoute53ResolverVPCID),
	})
	assertRoute53ResolverResourceAttributes(t, resource, ok, route53ResolverRuleAssociationResourceType, testRoute53ResolverRuleAssociationID,
		[]string{"rule_association", testRoute53ResolverRuleID, testRoute53ResolverVPCID, testRoute53ResolverRuleAssociationID},
		map[string]string{
			"name":             "example-association",
			"resolver_rule_id": testRoute53ResolverRuleID,
			"vpc_id":           testRoute53ResolverVPCID,
		})

	if _, ok := newRoute53ResolverRuleAssociationResource(&route53resolvertypes.ResolverRuleAssociation{Id: aws.String(testRoute53ResolverRuleAssociationID), ResolverRuleId: aws.String(testRoute53ResolverRuleID), Status: route53resolvertypes.ResolverRuleAssociationStatusFailed, VPCId: aws.String(testRoute53ResolverVPCID)}); ok {
		t.Fatal("failed resolver rule association should be skipped")
	}
	if _, ok := newRoute53ResolverRuleAssociationResource(&route53resolvertypes.ResolverRuleAssociation{Id: aws.String("rslvr-autodefined-rrassoc-vpc-1234567890abcdef0"), ResolverRuleId: aws.String(testRoute53ResolverRuleID), Status: route53resolvertypes.ResolverRuleAssociationStatusComplete, VPCId: aws.String(testRoute53ResolverVPCID)}); ok {
		t.Fatal("autodefined resolver rule association should be skipped")
	}
	if _, ok := newRoute53ResolverRuleAssociationResource(&route53resolvertypes.ResolverRuleAssociation{Id: aws.String(testRoute53ResolverRuleAssociationID), ResolverRuleId: aws.String("rslvr-autodefined-rr-internet-resolver"), Status: route53resolvertypes.ResolverRuleAssociationStatusComplete, VPCId: aws.String(testRoute53ResolverVPCID)}); ok {
		t.Fatal("association for an autodefined resolver rule should be skipped")
	}
}

func TestNewRoute53ResolverQueryLogResources(t *testing.T) {
	config, ok := newRoute53ResolverQueryLogConfigResource(&route53resolvertypes.ResolverQueryLogConfig{
		DestinationArn: aws.String("arn:aws:logs:us-east-1:123456789012:log-group:/resolver"),
		Id:             aws.String(testRoute53ResolverQueryLogConfigID),
		Name:           aws.String("resolver-queries"),
		ShareStatus:    route53resolvertypes.ShareStatusNotShared,
		Status:         route53resolvertypes.ResolverQueryLogConfigStatusCreated,
	})
	assertRoute53ResolverResourceAttributes(t, config, ok, route53ResolverQueryLogConfigResourceType, testRoute53ResolverQueryLogConfigID,
		[]string{"query_log_config", "resolver-queries", testRoute53ResolverQueryLogConfigID},
		map[string]string{"destination_arn": "arn:aws:logs:us-east-1:123456789012:log-group:/resolver", "name": "resolver-queries"})

	association, ok := newRoute53ResolverQueryLogConfigAssociationResource(&route53resolvertypes.ResolverQueryLogConfigAssociation{
		Id:                       aws.String(testRoute53ResolverQueryLogAssocID),
		ResolverQueryLogConfigId: aws.String(testRoute53ResolverQueryLogConfigID),
		ResourceId:               aws.String(testRoute53ResolverVPCID),
		Status:                   route53resolvertypes.ResolverQueryLogConfigAssociationStatusActive,
	})
	assertRoute53ResolverResourceAttributes(t, association, ok, route53ResolverQueryLogConfigAssociationResourceType, testRoute53ResolverQueryLogAssocID,
		[]string{"query_log_config_association", testRoute53ResolverQueryLogConfigID, testRoute53ResolverVPCID, testRoute53ResolverQueryLogAssocID},
		map[string]string{"resolver_query_log_config_id": testRoute53ResolverQueryLogConfigID, "resource_id": testRoute53ResolverVPCID})

	if _, ok := newRoute53ResolverQueryLogConfigResource(&route53resolvertypes.ResolverQueryLogConfig{DestinationArn: aws.String("arn"), Id: aws.String(testRoute53ResolverQueryLogConfigID), Name: aws.String("shared"), ShareStatus: route53resolvertypes.ShareStatusSharedWithMe, Status: route53resolvertypes.ResolverQueryLogConfigStatusCreated}); ok {
		t.Fatal("shared-with-me query log config should be skipped")
	}
	if _, ok := newRoute53ResolverQueryLogConfigAssociationResource(&route53resolvertypes.ResolverQueryLogConfigAssociation{Id: aws.String(testRoute53ResolverQueryLogAssocID), ResolverQueryLogConfigId: aws.String(testRoute53ResolverQueryLogConfigID), ResourceId: aws.String(testRoute53ResolverVPCID), Status: route53resolvertypes.ResolverQueryLogConfigAssociationStatusFailed}); ok {
		t.Fatal("failed query log association should be skipped")
	}
}

func TestNewRoute53ResolverFirewallResources(t *testing.T) {
	domainList, ok := newRoute53ResolverFirewallDomainListResource(&route53resolvertypes.FirewallDomainList{
		Id:     aws.String(testRoute53ResolverFirewallDomainID),
		Name:   aws.String("blocked-domains"),
		Status: route53resolvertypes.FirewallDomainListStatusComplete,
	}, []string{"example.com", "example.org"})
	assertRoute53ResolverResourceAttributes(t, domainList, ok, route53ResolverFirewallDomainListResourceType, testRoute53ResolverFirewallDomainID,
		[]string{"firewall_domain_list", "blocked-domains", testRoute53ResolverFirewallDomainID},
		map[string]string{"name": "blocked-domains", "domains.#": "2", "domains.0": "example.com", "domains.1": "example.org"})

	ruleGroup, ok := newRoute53ResolverFirewallRuleGroupResource(&route53resolvertypes.FirewallRuleGroup{
		Id:          aws.String(testRoute53ResolverFirewallRuleGroupID),
		Name:        aws.String("core-firewall"),
		ShareStatus: route53resolvertypes.ShareStatusNotShared,
		Status:      route53resolvertypes.FirewallRuleGroupStatusComplete,
	})
	assertRoute53ResolverResourceAttributes(t, ruleGroup, ok, route53ResolverFirewallRuleGroupResourceType, testRoute53ResolverFirewallRuleGroupID,
		[]string{"firewall_rule_group", "core-firewall", testRoute53ResolverFirewallRuleGroupID},
		map[string]string{"name": "core-firewall"})

	priority := int32(100)
	rule, ok := newRoute53ResolverFirewallRuleResource(route53resolvertypes.FirewallRule{
		Action:               route53resolvertypes.ActionBlock,
		BlockResponse:        route53resolvertypes.BlockResponseNodata,
		FirewallDomainListId: aws.String(testRoute53ResolverFirewallDomainID),
		FirewallRuleGroupId:  aws.String(testRoute53ResolverFirewallRuleGroupID),
		Name:                 aws.String("block-example"),
		Priority:             aws.Int32(priority),
	})
	assertRoute53ResolverResourceAttributes(t, rule, ok, route53ResolverFirewallRuleResourceType, testRoute53ResolverFirewallRuleGroupID+":"+testRoute53ResolverFirewallDomainID,
		[]string{"firewall_rule", testRoute53ResolverFirewallRuleGroupID, "block-example", testRoute53ResolverFirewallDomainID},
		map[string]string{
			"action":                  "BLOCK",
			"block_response":          "NODATA",
			"firewall_domain_list_id": testRoute53ResolverFirewallDomainID,
			"firewall_rule_group_id":  testRoute53ResolverFirewallRuleGroupID,
			"name":                    "block-example",
			"priority":                "100",
		})

	association, ok := newRoute53ResolverFirewallRuleGroupAssociationResource(&route53resolvertypes.FirewallRuleGroupAssociation{
		FirewallRuleGroupId: aws.String(testRoute53ResolverFirewallRuleGroupID),
		Id:                  aws.String(testRoute53ResolverFirewallAssocID),
		MutationProtection:  route53resolvertypes.MutationProtectionStatusEnabled,
		Name:                aws.String("core-vpc"),
		Priority:            aws.Int32(priority),
		Status:              route53resolvertypes.FirewallRuleGroupAssociationStatusComplete,
		VpcId:               aws.String(testRoute53ResolverVPCID),
	})
	assertRoute53ResolverResourceAttributes(t, association, ok, route53ResolverFirewallRuleGroupAssociationResourceType, testRoute53ResolverFirewallAssocID,
		[]string{"firewall_rule_group_association", testRoute53ResolverFirewallRuleGroupID, testRoute53ResolverVPCID, testRoute53ResolverFirewallAssocID},
		map[string]string{
			"firewall_rule_group_id": testRoute53ResolverFirewallRuleGroupID,
			"mutation_protection":    "ENABLED",
			"name":                   "core-vpc",
			"priority":               "100",
			"vpc_id":                 testRoute53ResolverVPCID,
		})

	if _, ok := newRoute53ResolverFirewallDomainListResource(&route53resolvertypes.FirewallDomainList{Id: aws.String(testRoute53ResolverFirewallDomainID), ManagedOwnerName: aws.String("Route 53 Resolver DNS Firewall"), Name: aws.String("aws-managed"), Status: route53resolvertypes.FirewallDomainListStatusComplete}, nil); ok {
		t.Fatal("AWS-managed firewall domain list should be skipped")
	}
	if _, ok := newRoute53ResolverFirewallRuleGroupResource(&route53resolvertypes.FirewallRuleGroup{Id: aws.String(testRoute53ResolverFirewallRuleGroupID), Name: aws.String("shared"), ShareStatus: route53resolvertypes.ShareStatusSharedWithMe, Status: route53resolvertypes.FirewallRuleGroupStatusComplete}); ok {
		t.Fatal("shared-with-me firewall rule group should be skipped")
	}
	if _, ok := newRoute53ResolverFirewallRuleResource(route53resolvertypes.FirewallRule{Action: route53resolvertypes.ActionBlock, FirewallRuleGroupId: aws.String(testRoute53ResolverFirewallRuleGroupID), Name: aws.String("missing-priority")}); ok {
		t.Fatal("firewall rule without priority should be skipped")
	}
	if _, ok := newRoute53ResolverFirewallRuleGroupAssociationResource(&route53resolvertypes.FirewallRuleGroupAssociation{FirewallRuleGroupId: aws.String(testRoute53ResolverFirewallRuleGroupID), Id: aws.String(testRoute53ResolverFirewallAssocID), ManagedOwnerName: aws.String("Firewall Manager"), Name: aws.String("managed"), Priority: aws.Int32(priority), Status: route53resolvertypes.FirewallRuleGroupAssociationStatusComplete, VpcId: aws.String(testRoute53ResolverVPCID)}); ok {
		t.Fatal("managed firewall rule group association should be skipped")
	}
}

func TestRoute53ResolverResourceNameAvoidsSanitizedCollisions(t *testing.T) {
	left := terraformutils.TfSanitize(route53ResolverResourceName("rule", "a_b", "c"))
	right := terraformutils.TfSanitize(route53ResolverResourceName("rule", "a", "b_c"))
	if left == right {
		t.Fatalf("resource names collide: %q", left)
	}
}

func TestRoute53ResolverResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "typed resource not found", err: &route53resolvertypes.ResourceNotFoundException{}, want: true},
		{name: "typed unknown resource", err: &route53resolvertypes.UnknownResourceException{}, want: true},
		{name: "generic resource not found", err: &smithy.GenericAPIError{Code: "ResourceNotFoundException"}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &route53resolvertypes.ResourceNotFoundException{}), want: true},
		{name: "access denied", err: &smithy.GenericAPIError{Code: "AccessDeniedException"}, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := route53ResolverResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("not found = %t, want %t", got, tt.want)
			}
		})
	}
}

func assertRoute53ResolverResourceAttributes(t *testing.T, resource terraformutils.Resource, ok bool, resourceType, resourceID string, nameParts []string, attributes map[string]string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, resourceType)
	}
	if resource.InstanceState.ID != resourceID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, resourceID)
	}
	for key, want := range attributes {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	wantName := terraformutils.TfSanitize(route53ResolverResourceName(nameParts...))
	if resource.ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
	}
}

type fakeRoute53ResolverConfigListClient struct {
	resolverConfigPages  []*route53resolver.ListResolverConfigsOutput
	resolverConfigInputs []*route53resolver.ListResolverConfigsInput
	dnssecConfigPages    []*route53resolver.ListResolverDnssecConfigsOutput
	dnssecConfigInputs   []*route53resolver.ListResolverDnssecConfigsInput
	firewallConfigPages  []*route53resolver.ListFirewallConfigsOutput
	firewallConfigInputs []*route53resolver.ListFirewallConfigsInput
}

func (c *fakeRoute53ResolverConfigListClient) ListResolverConfigs(_ context.Context, input *route53resolver.ListResolverConfigsInput, _ ...func(*route53resolver.Options)) (*route53resolver.ListResolverConfigsOutput, error) {
	c.resolverConfigInputs = append(c.resolverConfigInputs, input)
	page := c.resolverConfigPages[0]
	c.resolverConfigPages = c.resolverConfigPages[1:]
	return page, nil
}

func (c *fakeRoute53ResolverConfigListClient) ListResolverDnssecConfigs(_ context.Context, input *route53resolver.ListResolverDnssecConfigsInput, _ ...func(*route53resolver.Options)) (*route53resolver.ListResolverDnssecConfigsOutput, error) {
	c.dnssecConfigInputs = append(c.dnssecConfigInputs, input)
	page := c.dnssecConfigPages[0]
	c.dnssecConfigPages = c.dnssecConfigPages[1:]
	return page, nil
}

func (c *fakeRoute53ResolverConfigListClient) ListFirewallConfigs(_ context.Context, input *route53resolver.ListFirewallConfigsInput, _ ...func(*route53resolver.Options)) (*route53resolver.ListFirewallConfigsOutput, error) {
	c.firewallConfigInputs = append(c.firewallConfigInputs, input)
	page := c.firewallConfigPages[0]
	c.firewallConfigPages = c.firewallConfigPages[1:]
	return page, nil
}
