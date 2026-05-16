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
	for _, r := range resources {
		if r.InstanceInfo.Type != "aws_vpc_endpoint" {
			t.Errorf("expected type aws_vpc_endpoint, got %s", r.InstanceInfo.Type)
		}
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
