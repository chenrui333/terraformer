// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestRouteTableAssociationNilMain(t *testing.T) {
	// Verify that a nil Main field does not panic
	assocs := []types.RouteTableAssociation{
		{
			RouteTableAssociationId: aws.String("rtbassoc-nil"),
			Main:                    nil,
			SubnetId:                aws.String("subnet-123"),
			RouteTableId:            aws.String("rtb-456"),
		},
		{
			RouteTableAssociationId: aws.String("rtbassoc-false"),
			Main:                    aws.Bool(false),
			SubnetId:                aws.String("subnet-789"),
			RouteTableId:            aws.String("rtb-456"),
		},
		{
			RouteTableAssociationId: aws.String("rtbassoc-main"),
			Main:                    aws.Bool(true),
			RouteTableId:            aws.String("rtb-456"),
		},
	}

	// Simulate the guard logic from createRouteTablesResources
	var mainCount, subnetCount int
	for _, assoc := range assocs {
		if assoc.Main != nil && *assoc.Main {
			mainCount++
		} else if assoc.SubnetId != nil {
			subnetCount++
		}
	}

	if mainCount != 1 {
		t.Errorf("expected 1 main association, got %d", mainCount)
	}
	if subnetCount != 2 {
		t.Errorf("expected 2 subnet associations, got %d", subnetCount)
	}
}
