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

func (g *TransitGatewayGenerator) addTransitGatewayResource(tgw types.TransitGateway, localTGWs map[string]struct{}, addResource bool) {
	if tgw.State == types.TransitGatewayStateDeleted || tgw.State == types.TransitGatewayStateDeleting {
		return
	}
	transitGatewayID := StringValue(tgw.TransitGatewayId)
	if transitGatewayID == "" {
		return
	}
	localTGWs[transitGatewayID] = struct{}{}
	if !addResource {
		return
	}
	g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
		transitGatewayID,
		transitGatewayID,
		transitGatewayResourceType,
		"aws",
		tgwAllowEmptyValues,
	))
}

func (g *TransitGatewayGenerator) getTransitGateways(svc *ec2.Client, addResources bool) (map[string]struct{}, error) {
	localTGWs := make(map[string]struct{})
	p := ec2.NewDescribeTransitGatewaysPaginator(svc, &ec2.DescribeTransitGatewaysInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, tgw := range page.TransitGateways {
			g.addTransitGatewayResource(tgw, localTGWs, addResources)
		}
	}
	return localTGWs, nil
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

func (g *TransitGatewayGenerator) getTransitGatewayPeeringAttachments(svc *ec2.Client, localTGWs map[string]struct{}, loadAttachments, loadAccepters bool) error {
	p := ec2.NewDescribeTransitGatewayPeeringAttachmentsPaginator(svc, &ec2.DescribeTransitGatewayPeeringAttachmentsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, att := range page.TransitGatewayPeeringAttachments {
			resource, ok := newTransitGatewayPeeringAttachmentResource(att, localTGWs, loadAttachments, loadAccepters)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func newTransitGatewayPeeringAttachmentResource(att types.TransitGatewayPeeringAttachment, localTGWs map[string]struct{}, loadAttachments, loadAccepters bool) (terraformutils.Resource, bool) {
	if !transitGatewayAttachmentImportable(att.State) {
		return terraformutils.Resource{}, false
	}
	attachmentID := StringValue(att.TransitGatewayAttachmentId)
	if attachmentID == "" {
		return terraformutils.Resource{}, false
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
	if _, isLocal := localTGWs[requesterTGW]; isLocal && loadAttachments {
		resourceType = transitGatewayPeeringAttachmentResourceType
	} else if _, isLocal := localTGWs[accepterTGW]; isLocal && loadAccepters {
		resourceType = transitGatewayPeeringAttachmentAccepterType
	} else {
		return terraformutils.Resource{}, false
	}

	return terraformutils.NewSimpleResource(
		attachmentID,
		attachmentID,
		resourceType,
		"aws",
		tgwAllowEmptyValues,
	), true
}

func (g *TransitGatewayGenerator) getTransitGatewayRouteTableAssociations(svc *ec2.Client, loadAssociations, loadPropagations bool) error {
	if !loadAssociations && !loadPropagations {
		return nil
	}
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

			if loadAssociations && !isDefaultAssociation {
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

			if loadPropagations && !isDefaultPropagation {
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

func (g *TransitGatewayGenerator) getTransitGatewayMeteringPolicies(svc transitGatewayMeteringPolicyAPIClient, loadPolicies, loadEntries bool) error {
	if !loadPolicies && !loadEntries {
		return nil
	}
	policies, err := listTransitGatewayMeteringPolicies(svc)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		if resource, ok := newTransitGatewayMeteringPolicyResource(policy); ok {
			if loadPolicies {
				g.Resources = append(g.Resources, resource)
			}
			if loadEntries {
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
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayPolicyTables(svc *ec2.Client, loadTables, loadAssociations bool) error {
	if !loadTables && !loadAssociations {
		return nil
	}
	p := ec2.NewDescribeTransitGatewayPolicyTablesPaginator(svc, &ec2.DescribeTransitGatewayPolicyTablesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, table := range page.TransitGatewayPolicyTables {
			if resource, ok := newTransitGatewayPolicyTableResource(table); ok {
				if loadTables {
					g.Resources = append(g.Resources, resource)
				}
				if loadAssociations {
					if err := g.getTransitGatewayPolicyTableAssociations(svc, StringValue(table.TransitGatewayPolicyTableId)); err != nil {
						return err
					}
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

func (g *TransitGatewayGenerator) getTransitGatewayRouteTableAddOns(svc *ec2.Client, loadPrefixListReferences, loadRoutes bool) error {
	if !loadPrefixListReferences && !loadRoutes {
		return nil
	}
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
			if loadPrefixListReferences {
				if err := g.getTransitGatewayPrefixListReferences(svc, routeTableID); err != nil {
					return err
				}
			}
			if loadRoutes {
				if err := g.getTransitGatewayStaticRoutes(svc, routeTableID); err != nil {
					return err
				}
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
	resource := terraformutils.NewResource(
		id,
		transitGatewayResourceName("metering_policy", transitGatewayID, id),
		transitGatewayMeteringPolicyResourceType,
		"aws",
		attributes,
		tgwAllowEmptyValues,
		map[string]interface{}{},
	)
	setAwsFrameworkResourcePreserveIDAfterRefresh(&resource)
	return resource, true
}

func newTransitGatewayMeteringPolicyEntryResource(policyID string, entry types.TransitGatewayMeteringPolicyEntry) (terraformutils.Resource, bool) {
	ruleNumber := StringValue(entry.PolicyRuleNumber)
	if policyID == "" || ruleNumber == "" || !transitGatewayMeteringPolicyEntryPayerImportable(entry.MeteredAccount) || entry.State != types.TransitGatewayMeteringPolicyEntryStateAvailable {
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
	resource := terraformutils.NewResource(
		id,
		transitGatewayResourceName("metering_policy_entry", policyID, ruleNumber),
		transitGatewayMeteringPolicyEntryResourceType,
		"aws",
		attributes,
		tgwAllowEmptyValues,
		map[string]interface{}{},
	)
	setAwsFrameworkResourcePreserveIDAfterRefresh(&resource)
	return resource, true
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

func transitGatewayMeteringPolicyEntryPayerImportable(payer types.TransitGatewayMeteringPayerType) bool {
	switch payer {
	case types.TransitGatewayMeteringPayerTypeSourceAttachmentOwner,
		types.TransitGatewayMeteringPayerTypeDestinationAttachmentOwner,
		types.TransitGatewayMeteringPayerTypeTransitGatewayOwner:
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
		if !awsHasMorePages(page.NextToken) {
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
		if !awsHasMorePages(page.NextToken) {
			return entries, nil
		}
		input.NextToken = page.NextToken
	}
}

func (g *TransitGatewayGenerator) shouldLoadTransitGatewayResource(serviceNames ...string) bool {
	if !g.hasTransitGatewayTypedFilter() {
		return true
	}
	return shouldLoadAWSResourceForTypedFilters(g.Filter, serviceNames...)
}

func (g *TransitGatewayGenerator) hasTransitGatewayTypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			continue
		}
		serviceName := normalizeAWSFilterServiceName(filter.ServiceName)
		if serviceName == "ec2_transit_gateway" || strings.HasPrefix(serviceName, "ec2_transit_gateway_") {
			return true
		}
	}
	return false
}

// Generate TerraformResources from AWS API
func (g *TransitGatewayGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	g.Resources = []terraformutils.Resource{}

	loadTransitGateways := g.shouldLoadTransitGatewayResource(transitGatewayResourceType)
	loadPeeringAttachments := g.shouldLoadTransitGatewayResource(transitGatewayPeeringAttachmentResourceType)
	loadPeeringAttachmentAccepters := g.shouldLoadTransitGatewayResource(transitGatewayPeeringAttachmentAccepterType)
	loadPeeringResources := loadPeeringAttachments || loadPeeringAttachmentAccepters
	localTGWs := map[string]struct{}{}
	if loadTransitGateways || loadPeeringResources {
		var err error
		localTGWs, err = g.getTransitGateways(svc, loadTransitGateways)
		if err != nil {
			return err
		}
	}
	if g.shouldLoadTransitGatewayResource(transitGatewayRouteTableResourceType) {
		if err := g.getTransitGatewayRouteTables(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTransitGatewayResource(transitGatewayVpcAttachmentResourceType) {
		if err := g.getTransitGatewayVpcAttachments(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTransitGatewayResource(transitGatewayConnectResourceType) {
		if err := g.getTransitGatewayConnects(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTransitGatewayResource(transitGatewayConnectPeerResourceType) {
		if err := g.getTransitGatewayConnectPeers(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTransitGatewayResource(transitGatewayMulticastDomainResourceType) {
		if err := g.getTransitGatewayMulticastDomains(svc); err != nil {
			return err
		}
	}
	loadMeteringPolicies := g.shouldLoadTransitGatewayResource(transitGatewayMeteringPolicyResourceType)
	loadMeteringPolicyEntries := g.shouldLoadTransitGatewayResource(transitGatewayMeteringPolicyEntryResourceType)
	if loadMeteringPolicies || loadMeteringPolicyEntries {
		if err := g.getTransitGatewayMeteringPolicies(svc, loadMeteringPolicies, loadMeteringPolicyEntries); err != nil {
			return err
		}
	}
	if loadPeeringResources {
		if err := g.getTransitGatewayPeeringAttachments(svc, localTGWs, loadPeeringAttachments, loadPeeringAttachmentAccepters); err != nil {
			return err
		}
	}
	loadRouteTableAssociations := g.shouldLoadTransitGatewayResource(transitGatewayRouteTableAssociationResourceType)
	loadRouteTablePropagations := g.shouldLoadTransitGatewayResource(transitGatewayRouteTablePropagationResourceType)
	if loadRouteTableAssociations || loadRouteTablePropagations {
		if err := g.getTransitGatewayRouteTableAssociations(svc, loadRouteTableAssociations, loadRouteTablePropagations); err != nil {
			return err
		}
	}
	loadPolicyTables := g.shouldLoadTransitGatewayResource(transitGatewayPolicyTableResourceType)
	loadPolicyTableAssociations := g.shouldLoadTransitGatewayResource(transitGatewayPolicyTableAssociationType)
	if loadPolicyTables || loadPolicyTableAssociations {
		if err := g.getTransitGatewayPolicyTables(svc, loadPolicyTables, loadPolicyTableAssociations); err != nil {
			return err
		}
	}
	loadPrefixListReferences := g.shouldLoadTransitGatewayResource(transitGatewayPrefixListReferenceResourceType)
	loadRoutes := g.shouldLoadTransitGatewayResource(transitGatewayRouteResourceType)
	if loadPrefixListReferences || loadRoutes {
		if err := g.getTransitGatewayRouteTableAddOns(svc, loadPrefixListReferences, loadRoutes); err != nil {
			return err
		}
	}
	return nil
}
