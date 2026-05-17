// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestVpnGatewayFilterDeletedState(t *testing.T) {
	g := VpnGatewayGenerator{}
	output := &ec2.DescribeVpnGatewaysOutput{
		VpnGateways: []types.VpnGateway{
			{VpnGatewayId: aws.String("vgw-available"), State: types.VpnStateAvailable},
			{VpnGatewayId: aws.String("vgw-pending"), State: types.VpnStatePending},
			{VpnGatewayId: aws.String("vgw-deleted"), State: types.VpnStateDeleted},
			{VpnGatewayId: aws.String("vgw-deleting"), State: types.VpnStateDeleting},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	expectedIDs := map[string]bool{"vgw-available": false, "vgw-pending": false}
	for _, r := range resources {
		expectedIDs[r.InstanceState.ID] = true
	}
	for id, found := range expectedIDs {
		if !found {
			t.Errorf("expected resource %s not found", id)
		}
	}
}
