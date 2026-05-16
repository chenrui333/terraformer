// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestRouteTableAssociationNilMain(t *testing.T) {
	g := RouteTableGenerator{}
	tables := []types.RouteTable{
		{
			RouteTableId: aws.String("rtb-456"),
			VpcId:        aws.String("vpc-abc"),
			Associations: []types.RouteTableAssociation{
				{
					RouteTableAssociationId: aws.String("rtbassoc-nil"),
					Main:                    nil,
					SubnetId:                aws.String("subnet-123"),
				},
				{
					RouteTableAssociationId: aws.String("rtbassoc-false"),
					Main:                    aws.Bool(false),
					SubnetId:                aws.String("subnet-789"),
				},
				{
					RouteTableAssociationId: aws.String("rtbassoc-main"),
					Main:                    aws.Bool(true),
				},
			},
		},
	}

	resources := g.createResourcesFromTables(tables)

	// 1 route table + 1 main assoc + 2 subnet assocs = 4
	if len(resources) != 4 {
		t.Fatalf("expected 4 resources, got %d", len(resources))
	}
	if resources[0].InstanceInfo.Type != "aws_route_table" {
		t.Errorf("expected aws_route_table, got %s", resources[0].InstanceInfo.Type)
	}
	if resources[1].InstanceInfo.Type != "aws_route_table_association" {
		t.Errorf("expected aws_route_table_association for nil Main, got %s", resources[1].InstanceInfo.Type)
	}
	if resources[2].InstanceInfo.Type != "aws_route_table_association" {
		t.Errorf("expected aws_route_table_association for false Main, got %s", resources[2].InstanceInfo.Type)
	}
	if resources[3].InstanceInfo.Type != "aws_main_route_table_association" {
		t.Errorf("expected aws_main_route_table_association, got %s", resources[3].InstanceInfo.Type)
	}
}

func TestRouteTableGatewayAssociation(t *testing.T) {
	g := RouteTableGenerator{}
	tables := []types.RouteTable{
		{
			RouteTableId: aws.String("rtb-100"),
			Associations: []types.RouteTableAssociation{
				{
					RouteTableAssociationId: aws.String("rtbassoc-gw"),
					Main:                    aws.Bool(false),
					GatewayId:               aws.String("igw-999"),
				},
			},
		},
	}

	resources := g.createResourcesFromTables(tables)

	// 1 route table + 1 gateway assoc
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[1].InstanceInfo.Type != "aws_route_table_association" {
		t.Errorf("expected aws_route_table_association, got %s", resources[1].InstanceInfo.Type)
	}
	if resources[1].InstanceState.Attributes["gateway_id"] != "igw-999" {
		t.Errorf("expected gateway_id=igw-999, got %s", resources[1].InstanceState.Attributes["gateway_id"])
	}
}
