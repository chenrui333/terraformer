// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestCustomerGatewayFilterDeletedState(t *testing.T) {
	g := CustomerGatewayGenerator{}
	output := &ec2.DescribeCustomerGatewaysOutput{
		CustomerGateways: []types.CustomerGateway{
			{CustomerGatewayId: aws.String("cgw-available"), State: aws.String("available")},
			{CustomerGatewayId: aws.String("cgw-pending"), State: aws.String("pending")},
			{CustomerGatewayId: aws.String("cgw-deleted"), State: aws.String("deleted")},
			{CustomerGatewayId: aws.String("cgw-deleting"), State: aws.String("deleting")},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources (available+pending), got %d", len(resources))
	}

	expectedIDs := map[string]bool{
		"cgw-available": false,
		"cgw-pending":   false,
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

func TestCustomerGatewayNilState(t *testing.T) {
	g := CustomerGatewayGenerator{}
	output := &ec2.DescribeCustomerGatewaysOutput{
		CustomerGateways: []types.CustomerGateway{
			{CustomerGatewayId: aws.String("cgw-nostate"), State: nil},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource (nil state should pass), got %d", len(resources))
	}
}
