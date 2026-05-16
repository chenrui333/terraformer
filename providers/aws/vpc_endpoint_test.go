// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestVpcEndpointCreateResources(t *testing.T) {
	g := VpcEndpointGenerator{}
	output := &ec2.DescribeVpcEndpointsOutput{
		VpcEndpoints: []types.VpcEndpoint{
			{VpcEndpointId: aws.String("vpce-111"), State: types.StateAvailable},
			{VpcEndpointId: aws.String("vpce-222"), State: types.StateAvailable},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "vpce-111" {
		t.Errorf("expected vpce-111, got %s", resources[0].InstanceState.ID)
	}
	if resources[1].InstanceState.ID != "vpce-222" {
		t.Errorf("expected vpce-222, got %s", resources[1].InstanceState.ID)
	}
}

func TestVpcEndpointFilterDeletedState(t *testing.T) {
	g := VpcEndpointGenerator{}
	output := &ec2.DescribeVpcEndpointsOutput{
		VpcEndpoints: []types.VpcEndpoint{
			{VpcEndpointId: aws.String("vpce-alive"), State: types.StateAvailable},
			{VpcEndpointId: aws.String("vpce-dead"), State: types.StateDeleted},
			{VpcEndpointId: aws.String("vpce-dying"), State: types.StateDeleting},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].InstanceState.ID != "vpce-alive" {
		t.Errorf("expected vpce-alive, got %s", resources[0].InstanceState.ID)
	}
}

func TestVpcEndpointRouteTableAssociations(t *testing.T) {
	g := VpcEndpointGenerator{}
	output := &ec2.DescribeVpcEndpointsOutput{
		VpcEndpoints: []types.VpcEndpoint{
			{
				VpcEndpointId: aws.String("vpce-gw"),
				State:         types.StateAvailable,
				RouteTableIds: []string{"rtb-111", "rtb-222"},
			},
		},
	}

	resources := g.createResources(output)

	// 1 endpoint + 2 route table associations
	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}
	if resources[1].InstanceInfo.Type != "aws_vpc_endpoint_route_table_association" {
		t.Errorf("expected route_table_association, got %s", resources[1].InstanceInfo.Type)
	}
	if resources[1].InstanceState.ID != "vpce-gw/rtb-111" {
		t.Errorf("expected vpce-gw/rtb-111, got %s", resources[1].InstanceState.ID)
	}
	if resources[2].InstanceState.ID != "vpce-gw/rtb-222" {
		t.Errorf("expected vpce-gw/rtb-222, got %s", resources[2].InstanceState.ID)
	}
}

func TestVpcEndpointSubnetAssociations(t *testing.T) {
	g := VpcEndpointGenerator{}
	output := &ec2.DescribeVpcEndpointsOutput{
		VpcEndpoints: []types.VpcEndpoint{
			{
				VpcEndpointId: aws.String("vpce-iface"),
				State:         types.StateAvailable,
				SubnetIds:     []string{"subnet-aaa", "subnet-bbb"},
			},
		},
	}

	resources := g.createResources(output)

	// 1 endpoint + 2 subnet associations
	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}
	if resources[1].InstanceInfo.Type != "aws_vpc_endpoint_subnet_association" {
		t.Errorf("expected subnet_association, got %s", resources[1].InstanceInfo.Type)
	}
	if resources[1].InstanceState.ID != "vpce-iface/subnet-aaa" {
		t.Errorf("expected vpce-iface/subnet-aaa, got %s", resources[1].InstanceState.ID)
	}
}

func TestVpcEndpointAllowEmptyValues(t *testing.T) {
	g := VpcEndpointGenerator{}
	output := &ec2.DescribeVpcEndpointsOutput{
		VpcEndpoints: []types.VpcEndpoint{
			{VpcEndpointId: aws.String("vpce-111"), State: types.StateAvailable},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 1 {
		t.Fatal("expected 1 resource")
	}
	found := false
	for _, v := range resources[0].AllowEmptyValues {
		if v == "tags." {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected AllowEmptyValues to contain 'tags.'")
	}
}
