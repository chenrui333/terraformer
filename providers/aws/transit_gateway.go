// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var tgwAllowEmptyValues = []string{"tags."}

type TransitGatewayGenerator struct {
	AWSService
}

func (g *TransitGatewayGenerator) getTransitGateways(svc *ec2.Client) error {
	p := ec2.NewDescribeTransitGatewaysPaginator(svc, &ec2.DescribeTransitGatewaysInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, tgw := range page.TransitGateways {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(tgw.TransitGatewayId),
				StringValue(tgw.TransitGatewayId),
				"aws_ec2_transit_gateway",
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
				"aws_ec2_transit_gateway_route_table",
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
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(tgwa.TransitGatewayAttachmentId),
				StringValue(tgwa.TransitGatewayAttachmentId),
				"aws_ec2_transit_gateway_vpc_attachment",
				"aws",
				tgwAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *TransitGatewayGenerator) getTransitGatewayPeeringAttachments(svc *ec2.Client) error {
	p := ec2.NewDescribeTransitGatewayPeeringAttachmentsPaginator(svc, &ec2.DescribeTransitGatewayPeeringAttachmentsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, att := range page.TransitGatewayPeeringAttachments {
			if att.State == types.TransitGatewayAttachmentStateDeleted ||
				att.State == types.TransitGatewayAttachmentStateDeleting ||
				att.State == types.TransitGatewayAttachmentStateRejected ||
				att.State == types.TransitGatewayAttachmentStateRejecting ||
				att.State == types.TransitGatewayAttachmentStateFailed ||
				att.State == types.TransitGatewayAttachmentStateFailing {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(att.TransitGatewayAttachmentId),
				StringValue(att.TransitGatewayAttachmentId),
				"aws_ec2_transit_gateway_peering_attachment",
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
							"aws_ec2_transit_gateway_route_table_association",
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
							"aws_ec2_transit_gateway_route_table_propagation",
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
	if err := g.getTransitGatewayPeeringAttachments(svc); err != nil {
		return err
	}
	if err := g.getTransitGatewayRouteTableAssociations(svc); err != nil {
		return err
	}
	return nil
}
