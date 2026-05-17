// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestVpnConnectionFilterDeletedState(t *testing.T) {
	g := VpnConnectionGenerator{}
	output := &ec2.DescribeVpnConnectionsOutput{
		VpnConnections: []types.VpnConnection{
			{VpnConnectionId: aws.String("vpn-available"), State: types.VpnStateAvailable},
			{VpnConnectionId: aws.String("vpn-pending"), State: types.VpnStatePending},
			{VpnConnectionId: aws.String("vpn-deleted"), State: types.VpnStateDeleted},
			{VpnConnectionId: aws.String("vpn-deleting"), State: types.VpnStateDeleting},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 2 {
		t.Fatalf("expected 2 resources (available+pending), got %d", len(resources))
	}

	expectedIDs := map[string]bool{
		"vpn-available": false,
		"vpn-pending":   false,
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

func TestVpnConnectionEmptyOutput(t *testing.T) {
	g := VpnConnectionGenerator{}
	output := &ec2.DescribeVpnConnectionsOutput{
		VpnConnections: []types.VpnConnection{},
	}

	resources := g.createResources(output)

	if len(resources) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(resources))
	}
}
