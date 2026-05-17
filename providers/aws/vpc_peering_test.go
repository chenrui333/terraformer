// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestVpcPeeringFilterDeletedStates(t *testing.T) {
	g := VpcPeeringConnectionGenerator{}
	output := &ec2.DescribeVpcPeeringConnectionsOutput{
		VpcPeeringConnections: []types.VpcPeeringConnection{
			{
				VpcPeeringConnectionId: aws.String("pcx-active"),
				Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeActive},
			},
			{
				VpcPeeringConnectionId: aws.String("pcx-deleted"),
				Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeDeleted},
			},
			{
				VpcPeeringConnectionId: aws.String("pcx-rejected"),
				Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeRejected},
			},
			{
				VpcPeeringConnectionId: aws.String("pcx-expired"),
				Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeExpired},
			},
			{
				VpcPeeringConnectionId: aws.String("pcx-failed"),
				Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeFailed},
			},
			{
				VpcPeeringConnectionId: aws.String("pcx-pending"),
				Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodePendingAcceptance},
			},
			{
				VpcPeeringConnectionId: aws.String("pcx-provisioning"),
				Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeProvisioning},
			},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 3 {
		t.Fatalf("expected 3 resources (active+pending+provisioning), got %d", len(resources))
	}

	expectedIDs := map[string]bool{
		"pcx-active":       false,
		"pcx-pending":      false,
		"pcx-provisioning": false,
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

func TestVpcPeeringNilStatus(t *testing.T) {
	g := VpcPeeringConnectionGenerator{}
	output := &ec2.DescribeVpcPeeringConnectionsOutput{
		VpcPeeringConnections: []types.VpcPeeringConnection{
			{
				VpcPeeringConnectionId: aws.String("pcx-nostatus"),
				Status:                 nil,
			},
		},
	}

	resources := g.createResources(output)

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource (nil status should pass), got %d", len(resources))
	}
}
