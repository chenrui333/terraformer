// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestTransitGatewaySkipsDefaultRouteTable(t *testing.T) {
	g := TransitGatewayGenerator{}
	g.Resources = []terraformutils.Resource{}

	routeTables := []types.TransitGatewayRouteTable{
		{
			TransitGatewayRouteTableId:   aws.String("tgw-rtb-default"),
			DefaultAssociationRouteTable: aws.Bool(true),
			DefaultPropagationRouteTable: aws.Bool(true),
			TransitGatewayId:             aws.String("tgw-123"),
		},
		{
			TransitGatewayRouteTableId:   aws.String("tgw-rtb-custom"),
			DefaultAssociationRouteTable: aws.Bool(false),
			DefaultPropagationRouteTable: aws.Bool(false),
			TransitGatewayId:             aws.String("tgw-123"),
		},
	}

	for _, tgwrt := range routeTables {
		if *tgwrt.DefaultAssociationRouteTable {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			StringValue(tgwrt.TransitGatewayRouteTableId),
			StringValue(tgwrt.TransitGatewayRouteTableId),
			transitGatewayRouteTableResourceType,
			"aws",
			tgwAllowEmptyValues,
		))
	}

	if len(g.Resources) != 1 {
		t.Fatalf("expected 1 resource (skipping default), got %d", len(g.Resources))
	}
	if g.Resources[0].InstanceState.ID != "tgw-rtb-custom" {
		t.Errorf("expected tgw-rtb-custom, got %s", g.Resources[0].InstanceState.ID)
	}
}

func TestTransitGatewayPeeringAttachments(t *testing.T) {
	output := &ec2.DescribeTransitGatewayPeeringAttachmentsOutput{
		TransitGatewayPeeringAttachments: []types.TransitGatewayPeeringAttachment{
			{TransitGatewayAttachmentId: aws.String("tgw-attach-peer-1")},
			{TransitGatewayAttachmentId: aws.String("tgw-attach-peer-2")},
		},
	}

	var resources []terraformutils.Resource
	for _, att := range output.TransitGatewayPeeringAttachments {
		resources = append(resources, terraformutils.NewSimpleResource(
			StringValue(att.TransitGatewayAttachmentId),
			StringValue(att.TransitGatewayAttachmentId),
			transitGatewayPeeringAttachmentResourceType,
			"aws",
			tgwAllowEmptyValues,
		))
	}

	if len(resources) != 2 {
		t.Fatalf("expected 2 peering attachments, got %d", len(resources))
	}
	if resources[0].InstanceInfo.Type != transitGatewayPeeringAttachmentResourceType {
		t.Errorf("wrong resource type: %s", resources[0].InstanceInfo.Type)
	}
	if resources[0].InstanceState.ID != "tgw-attach-peer-1" {
		t.Errorf("expected tgw-attach-peer-1, got %s", resources[0].InstanceState.ID)
	}
}

func TestTransitGatewayAddOnResourceIDs(t *testing.T) {
	if got, want := transitGatewayCompositeID("tgw-rtb-123", "pl-123"), "tgw-rtb-123_pl-123"; got != want {
		t.Fatalf("transitGatewayCompositeID() = %q, want %q", got, want)
	}
	if got, want := transitGatewayMeteringPolicyEntryID("tgw-mp-123", "100"), "tgw-mp-123,100"; got != want {
		t.Fatalf("transitGatewayMeteringPolicyEntryID() = %q, want %q", got, want)
	}
}

func TestNewTransitGatewayConnectResource(t *testing.T) {
	resource, ok := newTransitGatewayConnectResource(types.TransitGatewayConnect{
		State:                               types.TransitGatewayAttachmentStateAvailable,
		TransitGatewayAttachmentId:          aws.String("tgw-attach-connect"),
		TransitGatewayId:                    aws.String("tgw-123"),
		TransportTransitGatewayAttachmentId: aws.String("tgw-attach-vpc"),
	})
	assertTransitGatewayResource(t, resource, ok, transitGatewayConnectResourceType, "tgw-attach-connect", map[string]string{
		"transit_gateway_id":      "tgw-123",
		"transport_attachment_id": "tgw-attach-vpc",
	})

	if _, ok := newTransitGatewayConnectResource(types.TransitGatewayConnect{
		State:                      types.TransitGatewayAttachmentStateDeleted,
		TransitGatewayAttachmentId: aws.String("tgw-attach-deleted"),
	}); ok {
		t.Fatal("deleted connect attachment should be skipped")
	}
}

func TestNewTransitGatewayConnectPeerResource(t *testing.T) {
	resource, ok := newTransitGatewayConnectPeerResource(types.TransitGatewayConnectPeer{
		State:                       types.TransitGatewayConnectPeerStateAvailable,
		TransitGatewayAttachmentId:  aws.String("tgw-attach-connect"),
		TransitGatewayConnectPeerId: aws.String("tgw-connect-peer-123"),
	})
	assertTransitGatewayResource(t, resource, ok, transitGatewayConnectPeerResourceType, "tgw-connect-peer-123", map[string]string{
		"transit_gateway_attachment_id": "tgw-attach-connect",
	})

	if _, ok := newTransitGatewayConnectPeerResource(types.TransitGatewayConnectPeer{
		State:                       types.TransitGatewayConnectPeerStateDeleting,
		TransitGatewayConnectPeerId: aws.String("tgw-connect-peer-deleting"),
	}); ok {
		t.Fatal("deleting connect peer should be skipped")
	}
}

func TestNewTransitGatewayPolicyAndRouteResources(t *testing.T) {
	policyTable, ok := newTransitGatewayPolicyTableResource(types.TransitGatewayPolicyTable{
		State:                       types.TransitGatewayPolicyTableStateAvailable,
		TransitGatewayId:            aws.String("tgw-123"),
		TransitGatewayPolicyTableId: aws.String("tgw-ptb-123"),
	})
	assertTransitGatewayResource(t, policyTable, ok, transitGatewayPolicyTableResourceType, "tgw-ptb-123", map[string]string{
		"transit_gateway_id": "tgw-123",
	})

	association, ok := newTransitGatewayPolicyTableAssociationResource(types.TransitGatewayPolicyTableAssociation{
		ResourceId:                  aws.String("vpc-123"),
		ResourceType:                types.TransitGatewayAttachmentResourceTypeVpc,
		State:                       types.TransitGatewayAssociationStateAssociated,
		TransitGatewayAttachmentId:  aws.String("tgw-attach-123"),
		TransitGatewayPolicyTableId: aws.String("tgw-ptb-123"),
	})
	assertTransitGatewayResource(t, association, ok, transitGatewayPolicyTableAssociationType, "tgw-ptb-123_tgw-attach-123", map[string]string{
		"resource_id":                     "vpc-123",
		"resource_type":                   "vpc",
		"transit_gateway_attachment_id":   "tgw-attach-123",
		"transit_gateway_policy_table_id": "tgw-ptb-123",
	})

	route, ok := newTransitGatewayRouteResource("tgw-rtb-123", types.TransitGatewayRoute{
		DestinationCidrBlock: aws.String("10.0.0.0/16"),
		State:                types.TransitGatewayRouteStateActive,
		Type:                 types.TransitGatewayRouteTypeStatic,
		TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{
			{TransitGatewayAttachmentId: aws.String("tgw-attach-123")},
		},
	})
	assertTransitGatewayResource(t, route, ok, transitGatewayRouteResourceType, "tgw-rtb-123_10.0.0.0/16", map[string]string{
		"destination_cidr_block":         "10.0.0.0/16",
		"transit_gateway_attachment_id":  "tgw-attach-123",
		"transit_gateway_route_table_id": "tgw-rtb-123",
	})

	if _, ok := newTransitGatewayRouteResource("tgw-rtb-123", types.TransitGatewayRoute{
		DestinationCidrBlock: aws.String("10.0.0.0/16"),
		State:                types.TransitGatewayRouteStateActive,
		Type:                 types.TransitGatewayRouteTypePropagated,
	}); ok {
		t.Fatal("propagated route should be skipped to avoid duplicate ownership with route table propagation")
	}
}

func TestNewTransitGatewayMulticastAndPrefixListResources(t *testing.T) {
	multicast, ok := newTransitGatewayMulticastDomainResource(types.TransitGatewayMulticastDomain{
		State:                           types.TransitGatewayMulticastDomainStateAvailable,
		TransitGatewayId:                aws.String("tgw-123"),
		TransitGatewayMulticastDomainId: aws.String("tgw-mcast-domain-123"),
	})
	assertTransitGatewayResource(t, multicast, ok, transitGatewayMulticastDomainResourceType, "tgw-mcast-domain-123", map[string]string{
		"transit_gateway_id": "tgw-123",
	})

	reference, ok := newTransitGatewayPrefixListReferenceResource(types.TransitGatewayPrefixListReference{
		PrefixListId:               aws.String("pl-123"),
		PrefixListOwnerId:          aws.String("123456789012"),
		State:                      types.TransitGatewayPrefixListReferenceStateAvailable,
		TransitGatewayRouteTableId: aws.String("tgw-rtb-123"),
		TransitGatewayAttachment: &types.TransitGatewayPrefixListAttachment{
			TransitGatewayAttachmentId: aws.String("tgw-attach-123"),
		},
	})
	assertTransitGatewayResource(t, reference, ok, transitGatewayPrefixListReferenceResourceType, "tgw-rtb-123_pl-123", map[string]string{
		"prefix_list_id":                 "pl-123",
		"prefix_list_owner_id":           "123456789012",
		"transit_gateway_attachment_id":  "tgw-attach-123",
		"transit_gateway_route_table_id": "tgw-rtb-123",
	})
}

func TestNewTransitGatewayMeteringResources(t *testing.T) {
	policy, ok := newTransitGatewayMeteringPolicyResource(types.TransitGatewayMeteringPolicy{
		MiddleboxAttachmentIds:         []string{"tgw-attach-mbox-1", "tgw-attach-mbox-2"},
		State:                          types.TransitGatewayMeteringPolicyStateAvailable,
		TransitGatewayId:               aws.String("tgw-123"),
		TransitGatewayMeteringPolicyId: aws.String("tgw-mp-123"),
	})
	assertTransitGatewayResource(t, policy, ok, transitGatewayMeteringPolicyResourceType, "tgw-mp-123", map[string]string{
		"middlebox_attachment_ids.#":         "2",
		"middlebox_attachment_ids.0":         "tgw-attach-mbox-1",
		"middlebox_attachment_ids.1":         "tgw-attach-mbox-2",
		"transit_gateway_id":                 "tgw-123",
		"transit_gateway_metering_policy_id": "tgw-mp-123",
	})

	entry, ok := newTransitGatewayMeteringPolicyEntryResource("tgw-mp-123", types.TransitGatewayMeteringPolicyEntry{
		MeteredAccount:   types.TransitGatewayMeteringPayerTypeDestinationAttachmentOwner,
		PolicyRuleNumber: aws.String("100"),
		State:            types.TransitGatewayMeteringPolicyEntryStateAvailable,
		MeteringPolicyRule: &types.TransitGatewayMeteringPolicyRule{
			DestinationCidrBlock:                    aws.String("10.0.0.0/16"),
			DestinationPortRange:                    aws.String("443-443"),
			DestinationTransitGatewayAttachmentId:   aws.String("tgw-attach-dst"),
			DestinationTransitGatewayAttachmentType: types.TransitGatewayAttachmentResourceTypeVpc,
			Protocol:                                aws.String("6"),
			SourceCidrBlock:                         aws.String("10.1.0.0/16"),
			SourcePortRange:                         aws.String("1024-65535"),
			SourceTransitGatewayAttachmentId:        aws.String("tgw-attach-src"),
			SourceTransitGatewayAttachmentType:      types.TransitGatewayAttachmentResourceTypeConnect,
		},
	})
	assertTransitGatewayResource(t, entry, ok, transitGatewayMeteringPolicyEntryResourceType, "tgw-mp-123,100", map[string]string{
		"destination_cidr_block":                      "10.0.0.0/16",
		"destination_port_range":                      "443-443",
		"destination_transit_gateway_attachment_id":   "tgw-attach-dst",
		"destination_transit_gateway_attachment_type": "vpc",
		"metered_account":                             "destination-attachment-owner",
		"policy_rule_number":                          "100",
		"protocol":                                    "6",
		"source_cidr_block":                           "10.1.0.0/16",
		"source_port_range":                           "1024-65535",
		"source_transit_gateway_attachment_id":        "tgw-attach-src",
		"source_transit_gateway_attachment_type":      "connect",
		"transit_gateway_metering_policy_id":          "tgw-mp-123",
	})

	if _, ok := newTransitGatewayMeteringPolicyResource(types.TransitGatewayMeteringPolicy{State: types.TransitGatewayMeteringPolicyStateDeleted, TransitGatewayId: aws.String("tgw-123"), TransitGatewayMeteringPolicyId: aws.String("tgw-mp-deleted")}); ok {
		t.Fatal("deleted metering policy should be skipped")
	}
	if _, ok := newTransitGatewayMeteringPolicyEntryResource("tgw-mp-123", types.TransitGatewayMeteringPolicyEntry{MeteredAccount: types.TransitGatewayMeteringPayerTypeTransitGatewayOwner, PolicyRuleNumber: aws.String("bad"), State: types.TransitGatewayMeteringPolicyEntryStateAvailable}); ok {
		t.Fatal("metering policy entry with nonnumeric rule number should be skipped")
	}
	if _, ok := newTransitGatewayMeteringPolicyEntryResource("tgw-mp-123", types.TransitGatewayMeteringPolicyEntry{MeteredAccount: types.TransitGatewayMeteringPayerTypeTransitGatewayOwner, PolicyRuleNumber: aws.String("200"), State: types.TransitGatewayMeteringPolicyEntryStateDeleted}); ok {
		t.Fatal("deleted metering policy entry should be skipped")
	}
}

func TestTransitGatewayConnectPagination(t *testing.T) {
	client := &fakeTransitGatewayConnectsClient{
		pages: []*ec2.DescribeTransitGatewayConnectsOutput{
			{
				NextToken: aws.String("next"),
				TransitGatewayConnects: []types.TransitGatewayConnect{
					{
						State:                      types.TransitGatewayAttachmentStateAvailable,
						TransitGatewayAttachmentId: aws.String("tgw-attach-1"),
					},
				},
			},
			{
				TransitGatewayConnects: []types.TransitGatewayConnect{
					{
						State:                      types.TransitGatewayAttachmentStateAvailable,
						TransitGatewayAttachmentId: aws.String("tgw-attach-2"),
					},
				},
			},
		},
	}
	g := TransitGatewayGenerator{}
	if err := g.getTransitGatewayConnects(client); err != nil {
		t.Fatalf("getTransitGatewayConnects() error = %v", err)
	}
	if client.calls != 2 {
		t.Fatalf("DescribeTransitGatewayConnects calls = %d, want 2", client.calls)
	}
	if len(g.Resources) != 2 {
		t.Fatalf("resources = %d, want 2", len(g.Resources))
	}
}

func TestTransitGatewayMeteringPolicyPagination(t *testing.T) {
	client := &fakeTransitGatewayMeteringPolicyClient{
		policyPages: []*ec2.DescribeTransitGatewayMeteringPoliciesOutput{
			{
				NextToken: aws.String("next-policy"),
				TransitGatewayMeteringPolicies: []types.TransitGatewayMeteringPolicy{
					{
						State:                          types.TransitGatewayMeteringPolicyStateAvailable,
						TransitGatewayId:               aws.String("tgw-123"),
						TransitGatewayMeteringPolicyId: aws.String("tgw-mp-1"),
					},
				},
			},
			{
				TransitGatewayMeteringPolicies: []types.TransitGatewayMeteringPolicy{
					{
						State:                          types.TransitGatewayMeteringPolicyStateDeleted,
						TransitGatewayId:               aws.String("tgw-123"),
						TransitGatewayMeteringPolicyId: aws.String("tgw-mp-deleted"),
					},
				},
			},
		},
		entryPagesByPolicy: map[string][]*ec2.GetTransitGatewayMeteringPolicyEntriesOutput{
			"tgw-mp-1": {
				{
					NextToken: aws.String("next-entry"),
					TransitGatewayMeteringPolicyEntries: []types.TransitGatewayMeteringPolicyEntry{
						{
							MeteredAccount:   types.TransitGatewayMeteringPayerTypeSourceAttachmentOwner,
							PolicyRuleNumber: aws.String("100"),
							State:            types.TransitGatewayMeteringPolicyEntryStateAvailable,
						},
					},
				},
				{
					TransitGatewayMeteringPolicyEntries: []types.TransitGatewayMeteringPolicyEntry{
						{
							MeteredAccount:   types.TransitGatewayMeteringPayerTypeTransitGatewayOwner,
							PolicyRuleNumber: aws.String("200"),
							State:            types.TransitGatewayMeteringPolicyEntryStateDeleted,
						},
					},
				},
			},
		},
	}
	g := TransitGatewayGenerator{}
	if err := g.getTransitGatewayMeteringPolicies(client); err != nil {
		t.Fatalf("getTransitGatewayMeteringPolicies() error = %v", err)
	}
	if client.policyCalls != 2 {
		t.Fatalf("DescribeTransitGatewayMeteringPolicies calls = %d, want 2", client.policyCalls)
	}
	if got, want := StringValue(client.policyInputs[1].NextToken), "next-policy"; got != want {
		t.Fatalf("policy next token = %q, want %q", got, want)
	}
	if client.entryCalls != 2 {
		t.Fatalf("GetTransitGatewayMeteringPolicyEntries calls = %d, want 2", client.entryCalls)
	}
	if got, want := StringValue(client.entryInputs[1].NextToken), "next-entry"; got != want {
		t.Fatalf("entry next token = %q, want %q", got, want)
	}
	if len(g.Resources) != 2 {
		t.Fatalf("resources = %d, want 2", len(g.Resources))
	}
}

func TestTransitGatewayStaticRoutePaginationFiltersAndSkips(t *testing.T) {
	client := &fakeTransitGatewayRoutesClient{
		pages: []*ec2.SearchTransitGatewayRoutesOutput{
			{
				NextToken: aws.String("next"),
				Routes: []types.TransitGatewayRoute{
					{
						DestinationCidrBlock: aws.String("10.0.0.0/16"),
						State:                types.TransitGatewayRouteStateActive,
						Type:                 types.TransitGatewayRouteTypeStatic,
					},
				},
			},
			{
				Routes: []types.TransitGatewayRoute{
					{
						DestinationCidrBlock: aws.String("10.1.0.0/16"),
						State:                types.TransitGatewayRouteStateActive,
						Type:                 types.TransitGatewayRouteTypePropagated,
					},
				},
			},
		},
	}
	g := TransitGatewayGenerator{}
	if err := g.getTransitGatewayStaticRoutes(client, "tgw-rtb-123"); err != nil {
		t.Fatalf("getTransitGatewayStaticRoutes() error = %v", err)
	}
	if client.calls != 2 {
		t.Fatalf("SearchTransitGatewayRoutes calls = %d, want 2", client.calls)
	}
	if len(client.filters) != 2 {
		t.Fatalf("filters = %d, want 2", len(client.filters))
	}
	if len(g.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(g.Resources))
	}
}

type fakeTransitGatewayMeteringPolicyClient struct {
	policyPages        []*ec2.DescribeTransitGatewayMeteringPoliciesOutput
	policyInputs       []*ec2.DescribeTransitGatewayMeteringPoliciesInput
	policyCalls        int
	entryPagesByPolicy map[string][]*ec2.GetTransitGatewayMeteringPolicyEntriesOutput
	entryInputs        []*ec2.GetTransitGatewayMeteringPolicyEntriesInput
	entryCalls         int
}

func (c *fakeTransitGatewayMeteringPolicyClient) DescribeTransitGatewayMeteringPolicies(_ context.Context, input *ec2.DescribeTransitGatewayMeteringPoliciesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayMeteringPoliciesOutput, error) {
	c.policyInputs = append(c.policyInputs, input)
	page := c.policyPages[c.policyCalls]
	c.policyCalls++
	return page, nil
}

func (c *fakeTransitGatewayMeteringPolicyClient) GetTransitGatewayMeteringPolicyEntries(_ context.Context, input *ec2.GetTransitGatewayMeteringPolicyEntriesInput, optFns ...func(*ec2.Options)) (*ec2.GetTransitGatewayMeteringPolicyEntriesOutput, error) {
	c.entryInputs = append(c.entryInputs, input)
	policyID := StringValue(input.TransitGatewayMeteringPolicyId)
	pages := c.entryPagesByPolicy[policyID]
	page := pages[0]
	c.entryPagesByPolicy[policyID] = pages[1:]
	c.entryCalls++
	return page, nil
}

type fakeTransitGatewayConnectsClient struct {
	pages []*ec2.DescribeTransitGatewayConnectsOutput
	calls int
}

func (c *fakeTransitGatewayConnectsClient) DescribeTransitGatewayConnects(context.Context, *ec2.DescribeTransitGatewayConnectsInput, ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayConnectsOutput, error) {
	page := c.pages[c.calls]
	c.calls++
	return page, nil
}

type fakeTransitGatewayRoutesClient struct {
	pages   []*ec2.SearchTransitGatewayRoutesOutput
	calls   int
	filters []types.Filter
}

func (c *fakeTransitGatewayRoutesClient) SearchTransitGatewayRoutes(_ context.Context, input *ec2.SearchTransitGatewayRoutesInput, _ ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
	if c.calls == 0 {
		c.filters = input.Filters
	}
	page := c.pages[c.calls]
	c.calls++
	return page, nil
}

func assertTransitGatewayResource(t *testing.T, resource terraformutils.Resource, ok bool, resourceType, id string, attributes map[string]string) {
	t.Helper()
	if !ok {
		t.Fatalf("expected %s resource", resourceType)
	}
	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("resource type = %q, want %s", resource.InstanceInfo.Type, resourceType)
	}
	if resource.InstanceState.ID != id {
		t.Fatalf("resource ID = %q, want %s", resource.InstanceState.ID, id)
	}
	for key, want := range attributes {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %s = %q, want %q", key, got, want)
		}
	}
}
