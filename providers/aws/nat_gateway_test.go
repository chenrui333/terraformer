// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestNatGatewayFilterDeletedState(t *testing.T) {
	g := NatGatewayGenerator{}
	output := &ec2.DescribeNatGatewaysOutput{
		NatGateways: []types.NatGateway{
			{
				NatGatewayId: aws.String("nat-available"),
				State:        types.NatGatewayStateAvailable,
			},
			{
				NatGatewayId: aws.String("nat-deleted"),
				State:        types.NatGatewayStateDeleted,
			},
			{
				NatGatewayId: aws.String("nat-deleting"),
				State:        types.NatGatewayStateDeleting,
			},
			{
				NatGatewayId: aws.String("nat-pending"),
				State:        types.NatGatewayStatePending,
			},
			{
				NatGatewayId: aws.String("nat-failed"),
				State:        types.NatGatewayStateFailed,
			},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}

	expectedIDs := map[string]bool{
		"nat-available": false,
		"nat-pending":   false,
		"nat-failed":    false,
	}
	for _, r := range resources {
		expectedIDs[r.InstanceState.ID] = true
	}
	for id, found := range expectedIDs {
		if !found {
			t.Errorf("expected resource %s not found", id)
		}
	}
}

func TestNatGatewayEmptyOutput(t *testing.T) {
	g := NatGatewayGenerator{}
	output := &ec2.DescribeNatGatewaysOutput{
		NatGateways: []types.NatGateway{},
	}

	resources := g.createResources(output)

	if len(resources) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(resources))
	}
}
