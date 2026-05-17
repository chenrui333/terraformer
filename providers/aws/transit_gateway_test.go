// SPDX-License-Identifier: Apache-2.0

package aws

import (
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
			"aws_ec2_transit_gateway_route_table",
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
			"aws_ec2_transit_gateway_peering_attachment",
			"aws",
			tgwAllowEmptyValues,
		))
	}

	if len(resources) != 2 {
		t.Fatalf("expected 2 peering attachments, got %d", len(resources))
	}
	if resources[0].InstanceInfo.Type != "aws_ec2_transit_gateway_peering_attachment" {
		t.Errorf("wrong resource type: %s", resources[0].InstanceInfo.Type)
	}
	if resources[0].InstanceState.ID != "tgw-attach-peer-1" {
		t.Errorf("expected tgw-attach-peer-1, got %s", resources[0].InstanceState.ID)
	}
}
