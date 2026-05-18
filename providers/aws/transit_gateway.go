// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var tgwAllowEmptyValues = []string{"tags."}

const (
	transitGatewayResourceType                      = "aws_ec2_transit_gateway"
	transitGatewayConnectResourceType               = "aws_ec2_transit_gateway_connect"
	transitGatewayConnectPeerResourceType           = "aws_ec2_transit_gateway_connect_peer"
	transitGatewayMeteringPolicyResourceType        = "aws_ec2_transit_gateway_metering_policy"
	transitGatewayMeteringPolicyEntryResourceType   = "aws_ec2_transit_gateway_metering_policy_entry"
	transitGatewayMulticastDomainResourceType       = "aws_ec2_transit_gateway_multicast_domain"
	transitGatewayPeeringAttachmentResourceType     = "aws_ec2_transit_gateway_peering_attachment"
	transitGatewayPeeringAttachmentAccepterType     = "aws_ec2_transit_gateway_peering_attachment_accepter"
	transitGatewayPolicyTableResourceType           = "aws_ec2_transit_gateway_policy_table"
	transitGatewayPolicyTableAssociationType        = "aws_ec2_transit_gateway_policy_table_association"
	transitGatewayPrefixListReferenceResourceType   = "aws_ec2_transit_gateway_prefix_list_reference"
	transitGatewayRouteResourceType                 = "aws_ec2_transit_gateway_route"
	transitGatewayRouteTableResourceType            = "aws_ec2_transit_gateway_route_table"
	transitGatewayRouteTableAssociationResourceType = "aws_ec2_transit_gateway_route_table_association"
	transitGatewayRouteTablePropagationResourceType = "aws_ec2_transit_gateway_route_table_propagation"
	transitGatewayVpcAttachmentResourceType         = "aws_ec2_transit_gateway_vpc_attachment"
	transitGatewayResourceIDSeparator               = "_"
	transitGatewayMeteringPolicyEntryIDSeparator    = ","
)

type TransitGatewayGenerator struct {
	AWSService
}

type transitGatewayMeteringPolicyAPIClient interface {
	DescribeTransitGatewayMeteringPolicies(context.Context, *ec2.DescribeTransitGatewayMeteringPoliciesInput, ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayMeteringPoliciesOutput, error)
	GetTransitGatewayMeteringPolicyEntries(context.Context, *ec2.GetTransitGatewayMeteringPolicyEntriesInput, ...func(*ec2.Options)) (*ec2.GetTransitGatewayMeteringPolicyEntriesOutput, error)
}

func (g *TransitGatewayGenerator) getTransitGateways(svc *ec2.Client) error {
	p := ec2.NewDescribeTransitGatewaysPaginator(svc, &ec2.DescribeTransitGatewaysInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, tgw := range page.TransitGateways {
			if tgw.State == types.TransitGatewayStateDeleted || tgw.State == types.TransitGatewayStateDeleting {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(tgw.TransitGatewayId),
				StringValue(tgw.TransitGatewayId),
				transitGatewayResourceType,
				"aws",
				tgwAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayRouteTables(svc *ec2.Client) error {
	p := ec2.NewDescribeTransitGatewayRouteTablesPaginator(svc, &ec2.DescribeTransitGatewayRouteTablesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, tgwrt := range page.TransitGatewayRouteTables {
			if tgwrt.DefaultAssociationRouteTable != nil && *tgwrt.DefaultAssociationRouteTable {
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
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayVpcAttachments(svc *ec2.Client) error {
	p := ec2.NewDescribeTransitGatewayVpcAttachmentsPaginator(svc, &ec2.DescribeTransitGatewayVpcAttachmentsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, tgwa := range page.TransitGatewayVpcAttachments {
			if !transitGatewayAttachmentImportable(tgwa.State) {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(tgwa.TransitGatewayAttachmentId),
				StringValue(tgwa.TransitGatewayAttachmentId),
				transitGatewayVpcAttachmentResourceType,
				"aws",
				tgwAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) localTGWIDs() map[string]struct{} {
	ids := make(map[string]struct{})
	for _, r := range g.Resources {
		if r.InstanceInfo.Type == transitGatewayResourceType {
			ids[r.InstanceState.ID] = struct{}{}
		}
	}
	return ids
}

func (g *TransitGatewayGenerator) getTransitGatewayPeeringAttachments(svc *ec2.Client) error {
	localTGWs := g.localTGWIDs()
	p := ec2.NewDescribeTransitGatewayPeeringAttachmentsPaginator(svc, &ec2.DescribeTransitGatewayPeeringAttachmentsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, att := range page.TransitGatewayPeeringAttachments {
			if !transitGatewayAttachmentImportable(att.State) {
				continue
			}

			requesterTGW := ""
			if att.RequesterTgwInfo != nil && att.RequesterTgwInfo.TransitGatewayId != nil {
				requesterTGW = *att.RequesterTgwInfo.TransitGatewayId
			}
			accepterTGW := ""
			if att.AccepterTgwInfo != nil && att.AccepterTgwInfo.TransitGatewayId != nil {
				accepterTGW = *att.AccepterTgwInfo.TransitGatewayId
			}

			resourceType := ""
			if _, isLocal := localTGWs[requesterTGW]; isLocal {
				resourceType = transitGatewayPeeringAttachmentResourceType
			} else if _, isLocal := localTGWs[accepterTGW]; isLocal {
				resourceType = transitGatewayPeeringAttachmentAccepterType
			} else {
				continue
			}

			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(att.TransitGatewayAttachmentId),
				StringValue(att.TransitGatewayAttachmentId),
				resourceType,
				"aws",
				tgwAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayRouteTableAssociations(svc *ec2.Client) error {
	rtPages := ec2.NewDescribeTransitGatewayRouteTablesPaginator(svc, &ec2.DescribeTransitGatewayRouteTablesInput{})
	for rtPages.HasMorePages() {
		rtPage, err := rtPages.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, rt := range rtPage.TransitGatewayRouteTables {
			if rt.State != types.TransitGatewayRouteTableStateAvailable {
				continue
			}
			isDefaultAssociation := rt.DefaultAssociationRouteTable != nil && *rt.DefaultAssociationRouteTable
			isDefaultPropagation := rt.DefaultPropagationRouteTable != nil && *rt.DefaultPropagationRouteTable

			if !isDefaultAssociation {
				assocPages := ec2.NewGetTransitGatewayRouteTableAssociationsPaginator(svc, &ec2.GetTransitGatewayRouteTableAssociationsInput{
					TransitGatewayRouteTableId: rt.TransitGatewayRouteTableId,
				})
				for assocPages.HasMorePages() {
					assocPage, err := assocPages.NextPage(context.TODO())
					if err != nil {
						return err
					}
					for _, assoc := range assocPage.Associations {
						if assoc.State != types.TransitGatewayAssociationStateAssociated {
							continue
						}
						id := StringValue(rt.TransitGatewayRouteTableId) + "_" + StringValue(assoc.TransitGatewayAttachmentId)
						g.Resources = append(g.Resources, terraformutils.NewResource(
							id,
							id,
							transitGatewayRouteTableAssociationResourceType,
							"aws",
							map[string]string{
								"transit_gateway_attachment_id":  StringValue(assoc.TransitGatewayAttachmentId),
								"transit_gateway_route_table_id": StringValue(rt.TransitGatewayRouteTableId),
							},
							tgwAllowEmptyValues,
							map[string]interface{}{},
						))
					}
				}
			}

			if !isDefaultPropagation {
				propPages := ec2.NewGetTransitGatewayRouteTablePropagationsPaginator(svc, &ec2.GetTransitGatewayRouteTablePropagationsInput{
					TransitGatewayRouteTableId: rt.TransitGatewayRouteTableId,
				})
				for propPages.HasMorePages() {
					propPage, err := propPages.NextPage(context.TODO())
					if err != nil {
						return err
					}
					for _, prop := range propPage.TransitGatewayRouteTablePropagations {
						if prop.State != types.TransitGatewayPropagationStateEnabled {
							continue
						}
						id := StringValue(rt.TransitGatewayRouteTableId) + "_" + StringValue(prop.TransitGatewayAttachmentId)
						g.Resources = append(g.Resources, terraformutils.NewResource(
							id,
							id,
							transitGatewayRouteTablePropagationResourceType,
							"aws",
							map[string]string{
								"transit_gateway_attachment_id":  StringValue(prop.TransitGatewayAttachmentId),
								"transit_gateway_route_table_id": StringValue(rt.TransitGatewayRouteTableId),
							},
							tgwAllowEmptyValues,
							map[string]interface{}{},
						))
					}
				}
			}
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayConnects(svc ec2.DescribeTransitGatewayConnectsAPIClient) error {
	p := ec2.NewDescribeTransitGatewayConnectsPaginator(svc, &ec2.DescribeTransitGatewayConnectsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, connect := range page.TransitGatewayConnects {
			if resource, ok := newTransitGatewayConnectResource(connect); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayConnectPeers(svc ec2.DescribeTransitGatewayConnectPeersAPIClient) error {
	p := ec2.NewDescribeTransitGatewayConnectPeersPaginator(svc, &ec2.DescribeTransitGatewayConnectPeersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, peer := range page.TransitGatewayConnectPeers {
			if resource, ok := newTransitGatewayConnectPeerResource(peer); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayMulticastDomains(svc ec2.DescribeTransitGatewayMulticastDomainsAPIClient) error {
	p := ec2.NewDescribeTransitGatewayMulticastDomainsPaginator(svc, &ec2.DescribeTransitGatewayMulticastDomainsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, domain := range page.TransitGatewayMulticastDomains {
			if resource, ok := newTransitGatewayMulticastDomainResource(domain); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayMeteringPolicies(svc transitGatewayMeteringPolicyAPIClient) error {
	policies, err := listTransitGatewayMeteringPolicies(svc)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		if resource, ok := newTransitGatewayMeteringPolicyResource(policy); ok {
			g.Resources = append(g.Resources, resource)
			entries, err := listTransitGatewayMeteringPolicyEntries(svc, StringValue(policy.TransitGatewayMeteringPolicyId))
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if resource, ok := newTransitGatewayMeteringPolicyEntryResource(StringValue(policy.TransitGatewayMeteringPolicyId), entry); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayPolicyTables(svc *ec2.Client) error {
	p := ec2.NewDescribeTransitGatewayPolicyTablesPaginator(svc, &ec2.DescribeTransitGatewayPolicyTablesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, table := range page.TransitGatewayPolicyTables {
			if resource, ok := newTransitGatewayPolicyTableResource(table); ok {
				g.Resources = append(g.Resources, resource)
				if err := g.getTransitGatewayPolicyTableAssociations(svc, StringValue(table.TransitGatewayPolicyTableId)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayPolicyTableAssociations(svc *ec2.Client, policyTableID string) error {
	if policyTableID == "" {
		return nil
	}
	p := ec2.NewGetTransitGatewayPolicyTableAssociationsPaginator(svc, &ec2.GetTransitGatewayPolicyTableAssociationsInput{
		TransitGatewayPolicyTableId: aws.String(policyTableID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, association := range page.Associations {
			if resource, ok := newTransitGatewayPolicyTableAssociationResource(association); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayRouteTableAddOns(svc *ec2.Client) error {
	rtPages := ec2.NewDescribeTransitGatewayRouteTablesPaginator(svc, &ec2.DescribeTransitGatewayRouteTablesInput{})
	for rtPages.HasMorePages() {
		rtPage, err := rtPages.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, rt := range rtPage.TransitGatewayRouteTables {
			if rt.State != types.TransitGatewayRouteTableStateAvailable {
				continue
			}
			routeTableID := StringValue(rt.TransitGatewayRouteTableId)
			if err := g.getTransitGatewayPrefixListReferences(svc, routeTableID); err != nil {
				return err
			}
			if err := g.getTransitGatewayStaticRoutes(svc, routeTableID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayPrefixListReferences(svc *ec2.Client, routeTableID string) error {
	if routeTableID == "" {
		return nil
	}
	p := ec2.NewGetTransitGatewayPrefixListReferencesPaginator(svc, &ec2.GetTransitGatewayPrefixListReferencesInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, reference := range page.TransitGatewayPrefixListReferences {
			if resource, ok := newTransitGatewayPrefixListReferenceResource(reference); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayStaticRoutes(svc ec2.SearchTransitGatewayRoutesAPIClient, routeTableID string) error {
	if routeTableID == "" {
		return nil
	}
	p := ec2.NewSearchTransitGatewayRoutesPaginator(svc, &ec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		Filters: []types.Filter{
			{Name: aws.String("type"), Values: []string{string(types.TransitGatewayRouteTypeStatic)}},
			{Name: aws.String("state"), Values: []string{string(types.TransitGatewayRouteStateActive), string(types.TransitGatewayRouteStateBlackhole)}},
		},
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, route := range page.Routes {
			if resource, ok := newTransitGatewayRouteResource(routeTableID, route); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newTransitGatewayConnectResource(connect types.TransitGatewayConnect) (terraformutils.Resource, bool) {
	id := StringValue(connect.TransitGatewayAttachmentId)
	if id == "" || !transitGatewayAttachmentImportable(connect.State) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		transitGatewayResourceName("connect", StringValue(connect.TransitGatewayId), id),
		transitGatewayConnectResourceType,
		"aws",
		map[string]string{
			"transit_gateway_id":      StringValue(connect.TransitGatewayId),
			"transport_attachment_id": StringValue(connect.TransportTransitGatewayAttachmentId),
		},
		tgwAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newTransitGatewayConnectPeerResource(peer types.TransitGatewayConnectPeer) (terraformutils.Resource, bool) {
	id := StringValue(peer.TransitGatewayConnectPeerId)
	if id == "" || !transitGatewayConnectPeerImportable(peer.State) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		transitGatewayResourceName("connect_peer", StringValue(peer.TransitGatewayAttachmentId), id),
		transitGatewayConnectPeerResourceType,
		"aws",
		map[string]string{
			"transit_gateway_attachment_id": StringValue(peer.TransitGatewayAttachmentId),
		},
		tgwAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newTransitGatewayMulticastDomainResource(domain types.TransitGatewayMulticastDomain) (terraformutils.Resource, bool) {
	id := StringValue(domain.TransitGatewayMulticastDomainId)
	if id == "" || !transitGatewayMulticastDomainImportable(domain.State) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		transitGatewayResourceName("multicast_domain", StringValue(domain.TransitGatewayId), id),
		transitGatewayMulticastDomainResourceType,
		"aws",
		map[string]string{
			"transit_gateway_id": StringValue(domain.TransitGatewayId),
		},
		tgwAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newTransitGatewayMeteringPolicyResource(policy types.TransitGatewayMeteringPolicy) (terraformutils.Resource, bool) {
	id := StringValue(policy.TransitGatewayMeteringPolicyId)
	transitGatewayID := StringValue(policy.TransitGatewayId)
	if id == "" || transitGatewayID == "" || !transitGatewayMeteringPolicyImportable(policy.State) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"transit_gateway_id":                 transitGatewayID,
		"transit_gateway_metering_policy_id": id,
	}
	putTransitGatewayStringListAttributes(attributes, "middlebox_attachment_ids", policy.MiddleboxAttachmentIds)
	return terraformutils.NewResource(
		id,
		transitGatewayResourceName("metering_policy", transitGatewayID, id),
		transitGatewayMeteringPolicyResourceType,
		"aws",
		attributes,
		tgwAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newTransitGatewayMeteringPolicyEntryResource(policyID string, entry types.TransitGatewayMeteringPolicyEntry) (terraformutils.Resource, bool) {
	ruleNumber := StringValue(entry.PolicyRuleNumber)
	if policyID == "" || ruleNumber == "" || entry.MeteredAccount == "" || entry.State != types.TransitGatewayMeteringPolicyEntryStateAvailable {
		return terraformutils.Resource{}, false
	}
	if _, err := strconv.ParseInt(ruleNumber, 10, 64); err != nil {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"metered_account":                    string(entry.MeteredAccount),
		"policy_rule_number":                 ruleNumber,
		"transit_gateway_metering_policy_id": policyID,
	}
	if entry.MeteringPolicyRule != nil {
		putTransitGatewayString(attributes, "destination_cidr_block", StringValue(entry.MeteringPolicyRule.DestinationCidrBlock))
		putTransitGatewayString(attributes, "destination_port_range", StringValue(entry.MeteringPolicyRule.DestinationPortRange))
		putTransitGatewayString(attributes, "destination_transit_gateway_attachment_id", StringValue(entry.MeteringPolicyRule.DestinationTransitGatewayAttachmentId))
		putTransitGatewayString(attributes, "destination_transit_gateway_attachment_type", string(entry.MeteringPolicyRule.DestinationTransitGatewayAttachmentType))
		putTransitGatewayString(attributes, "protocol", StringValue(entry.MeteringPolicyRule.Protocol))
		putTransitGatewayString(attributes, "source_cidr_block", StringValue(entry.MeteringPolicyRule.SourceCidrBlock))
		putTransitGatewayString(attributes, "source_port_range", StringValue(entry.MeteringPolicyRule.SourcePortRange))
		putTransitGatewayString(attributes, "source_transit_gateway_attachment_id", StringValue(entry.MeteringPolicyRule.SourceTransitGatewayAttachmentId))
		putTransitGatewayString(attributes, "source_transit_gateway_attachment_type", string(entry.MeteringPolicyRule.SourceTransitGatewayAttachmentType))
	}
	id := transitGatewayMeteringPolicyEntryID(policyID, ruleNumber)
	return terraformutils.NewResource(
		id,
		transitGatewayResourceName("metering_policy_entry", policyID, ruleNumber),
		transitGatewayMeteringPolicyEntryResourceType,
		"aws",
		attributes,
		tgwAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newTransitGatewayPolicyTableResource(table types.TransitGatewayPolicyTable) (terraformutils.Resource, bool) {
	id := StringValue(table.TransitGatewayPolicyTableId)
	if id == "" || !transitGatewayPolicyTableImportable(table.State) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		transitGatewayResourceName("policy_table", StringValue(table.TransitGatewayId), id),
		transitGatewayPolicyTableResourceType,
		"aws",
		map[string]string{
			"transit_gateway_id": StringValue(table.TransitGatewayId),
		},
		tgwAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newTransitGatewayPolicyTableAssociationResource(association types.TransitGatewayPolicyTableAssociation) (terraformutils.Resource, bool) {
	policyTableID := StringValue(association.TransitGatewayPolicyTableId)
	attachmentID := StringValue(association.TransitGatewayAttachmentId)
	if policyTableID == "" || attachmentID == "" || association.State != types.TransitGatewayAssociationStateAssociated {
		return terraformutils.Resource{}, false
	}
	id := transitGatewayCompositeID(policyTableID, attachmentID)
	return terraformutils.NewResource(
		id,
		transitGatewayResourceName("policy_table_association", policyTableID, attachmentID),
		transitGatewayPolicyTableAssociationType,
		"aws",
		map[string]string{
			"resource_id":                     StringValue(association.ResourceId),
			"resource_type":                   string(association.ResourceType),
			"transit_gateway_attachment_id":   attachmentID,
			"transit_gateway_policy_table_id": policyTableID,
		},
		tgwAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newTransitGatewayPrefixListReferenceResource(reference types.TransitGatewayPrefixListReference) (terraformutils.Resource, bool) {
	routeTableID := StringValue(reference.TransitGatewayRouteTableId)
	prefixListID := StringValue(reference.PrefixListId)
	if routeTableID == "" || prefixListID == "" || !transitGatewayPrefixListReferenceImportable(reference.State) {
		return terraformutils.Resource{}, false
	}
	id := transitGatewayCompositeID(routeTableID, prefixListID)
	attributes := map[string]string{
		"prefix_list_id":                 prefixListID,
		"prefix_list_owner_id":           StringValue(reference.PrefixListOwnerId),
		"transit_gateway_route_table_id": routeTableID,
	}
	if reference.TransitGatewayAttachment != nil {
		attributes["transit_gateway_attachment_id"] = StringValue(reference.TransitGatewayAttachment.TransitGatewayAttachmentId)
	}
	return terraformutils.NewResource(
		id,
		transitGatewayResourceName("prefix_list_reference", routeTableID, prefixListID),
		transitGatewayPrefixListReferenceResourceType,
		"aws",
		attributes,
		tgwAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newTransitGatewayRouteResource(routeTableID string, route types.TransitGatewayRoute) (terraformutils.Resource, bool) {
	destination := StringValue(route.DestinationCidrBlock)
	if routeTableID == "" || destination == "" || !transitGatewayRouteImportable(route) {
		return terraformutils.Resource{}, false
	}
	id := transitGatewayCompositeID(routeTableID, destination)
	attributes := map[string]string{
		"destination_cidr_block":         destination,
		"transit_gateway_route_table_id": routeTableID,
	}
	if len(route.TransitGatewayAttachments) > 0 {
		attributes["transit_gateway_attachment_id"] = StringValue(route.TransitGatewayAttachments[0].TransitGatewayAttachmentId)
	}
	return terraformutils.NewResource(
		id,
		transitGatewayResourceName("route", routeTableID, destination),
		transitGatewayRouteResourceType,
		"aws",
		attributes,
		tgwAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func transitGatewayAttachmentImportable(state types.TransitGatewayAttachmentState) bool {
	switch state {
	case types.TransitGatewayAttachmentStateDeleting,
		types.TransitGatewayAttachmentStateDeleted,
		types.TransitGatewayAttachmentStateRejecting,
		types.TransitGatewayAttachmentStateRejected,
		types.TransitGatewayAttachmentStateFailing,
		types.TransitGatewayAttachmentStateFailed:
		return false
	default:
		return true
	}
}

func transitGatewayConnectPeerImportable(state types.TransitGatewayConnectPeerState) bool {
	switch state {
	case types.TransitGatewayConnectPeerStateDeleting,
		types.TransitGatewayConnectPeerStateDeleted:
		return false
	default:
		return true
	}
}

func transitGatewayMulticastDomainImportable(state types.TransitGatewayMulticastDomainState) bool {
	switch state {
	case types.TransitGatewayMulticastDomainStateDeleting,
		types.TransitGatewayMulticastDomainStateDeleted:
		return false
	default:
		return true
	}
}

func transitGatewayMeteringPolicyImportable(state types.TransitGatewayMeteringPolicyState) bool {
	return state == types.TransitGatewayMeteringPolicyStateAvailable
}

func transitGatewayPolicyTableImportable(state types.TransitGatewayPolicyTableState) bool {
	switch state {
	case types.TransitGatewayPolicyTableStateDeleting,
		types.TransitGatewayPolicyTableStateDeleted:
		return false
	default:
		return true
	}
}

func transitGatewayPrefixListReferenceImportable(state types.TransitGatewayPrefixListReferenceState) bool {
	return state != types.TransitGatewayPrefixListReferenceStateDeleting
}

func transitGatewayRouteImportable(route types.TransitGatewayRoute) bool {
	if route.Type != types.TransitGatewayRouteTypeStatic {
		return false
	}
	switch route.State {
	case types.TransitGatewayRouteStateActive, types.TransitGatewayRouteStateBlackhole:
		return true
	default:
		return false
	}
}

func transitGatewayCompositeID(parts ...string) string {
	return strings.Join(parts, transitGatewayResourceIDSeparator)
}

func transitGatewayMeteringPolicyEntryID(policyID, ruleNumber string) string {
	return strings.Join([]string{policyID, ruleNumber}, transitGatewayMeteringPolicyEntryIDSeparator)
}

func transitGatewayResourceName(parts ...string) string {
	return awsResourceNameWithLengths(parts...)
}

func putTransitGatewayString(attributes map[string]string, key, value string) {
	if value != "" {
		attributes[key] = value
	}
}

func putTransitGatewayStringListAttributes(attributes map[string]string, key string, values []string) {
	attributes[key+".#"] = strconv.Itoa(len(values))
	for i, value := range values {
		attributes[key+"."+strconv.Itoa(i)] = value
	}
}

func listTransitGatewayMeteringPolicies(svc transitGatewayMeteringPolicyAPIClient) ([]types.TransitGatewayMeteringPolicy, error) {
	input := &ec2.DescribeTransitGatewayMeteringPoliciesInput{}
	var policies []types.TransitGatewayMeteringPolicy
	for {
		page, err := svc.DescribeTransitGatewayMeteringPolicies(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		policies = append(policies, page.TransitGatewayMeteringPolicies...)
		if page.NextToken == nil {
			return policies, nil
		}
		input.NextToken = page.NextToken
	}
}

func listTransitGatewayMeteringPolicyEntries(svc transitGatewayMeteringPolicyAPIClient, policyID string) ([]types.TransitGatewayMeteringPolicyEntry, error) {
	if policyID == "" {
		return nil, nil
	}
	input := &ec2.GetTransitGatewayMeteringPolicyEntriesInput{
		TransitGatewayMeteringPolicyId: aws.String(policyID),
	}
	var entries []types.TransitGatewayMeteringPolicyEntry
	for {
		page, err := svc.GetTransitGatewayMeteringPolicyEntries(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		entries = append(entries, page.TransitGatewayMeteringPolicyEntries...)
		if page.NextToken == nil {
			return entries, nil
		}
		input.NextToken = page.NextToken
	}
}

// Generate TerraformResources from AWS API
func (g *TransitGatewayGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	g.Resources = []terraformutils.Resource{}

	if err := g.getTransitGateways(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayRouteTables(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayVpcAttachments(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayConnects(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayConnectPeers(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayMulticastDomains(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayMeteringPolicies(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayPeeringAttachments(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayRouteTableAssociations(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayPolicyTables(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayRouteTableAddOns(svc); err != nil {
		return err
	}
	return nil
}
