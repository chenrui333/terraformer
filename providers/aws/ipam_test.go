// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewIPAMResource(t *testing.T) {
	resource, ok := newIPAMResource(types.Ipam{
		IpamId:     aws.String("ipam-123"),
		IpamRegion: aws.String("us-east-1"),
		State:      types.IpamStateCreateComplete,
	})
	if !ok {
		t.Fatal("newIPAMResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ipamResourceType {
		t.Fatalf("resource type = %q, want %q", got, ipamResourceType)
	}
	if got := resource.InstanceState.ID; got != "ipam-123" {
		t.Fatalf("resource ID = %q, want ipam-123", got)
	}

	if _, ok := newIPAMResource(types.Ipam{State: types.IpamStateCreateComplete}); ok {
		t.Fatal("IPAM with empty ID should be skipped")
	}
	if _, ok := newIPAMResource(types.Ipam{
		IpamId: aws.String("ipam-123"),
		State:  types.IpamStateDeleteInProgress,
	}); ok {
		t.Fatal("delete-in-progress IPAM should be skipped")
	}
}

func TestNewIPAMScopeResource(t *testing.T) {
	resource, ok := newIPAMScopeResource(types.IpamScope{
		IpamArn:       aws.String("arn:aws:ec2::123456789012:ipam/ipam-123"),
		IpamScopeId:   aws.String("ipam-scope-123"),
		IpamScopeType: types.IpamScopeTypePrivate,
		IsDefault:     aws.Bool(false),
		State:         types.IpamScopeStateModifyComplete,
	})
	if !ok {
		t.Fatal("newIPAMScopeResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ipamScopeResourceType {
		t.Fatalf("resource type = %q, want %q", got, ipamScopeResourceType)
	}
	if got := resource.InstanceState.ID; got != "ipam-scope-123" {
		t.Fatalf("resource ID = %q, want ipam-scope-123", got)
	}

	if _, ok := newIPAMScopeResource(types.IpamScope{
		IpamScopeId: aws.String("ipam-scope-123"),
		IsDefault:   aws.Bool(true),
		State:       types.IpamScopeStateCreateComplete,
	}); ok {
		t.Fatal("default IPAM scope should be skipped")
	}
	if _, ok := newIPAMScopeResource(types.IpamScope{
		IpamScopeId: aws.String("ipam-scope-123"),
		State:       types.IpamScopeStateDeleteComplete,
	}); ok {
		t.Fatal("delete-complete IPAM scope should be skipped")
	}
}

func TestNewIPAMPoolResource(t *testing.T) {
	resource, ok := newIPAMPoolResource(types.IpamPool{
		AddressFamily: types.AddressFamilyIpv4,
		IpamPoolId:    aws.String("ipam-pool-123"),
		IpamScopeArn:  aws.String("arn:aws:ec2::123456789012:ipam-scope/ipam-scope-123"),
		State:         types.IpamPoolStateCreateComplete,
	})
	if !ok {
		t.Fatal("newIPAMPoolResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ipamPoolResourceType {
		t.Fatalf("resource type = %q, want %q", got, ipamPoolResourceType)
	}
	if got := resource.InstanceState.ID; got != "ipam-pool-123" {
		t.Fatalf("resource ID = %q, want ipam-pool-123", got)
	}

	if _, ok := newIPAMPoolResource(types.IpamPool{State: types.IpamPoolStateCreateComplete}); ok {
		t.Fatal("IPAM pool with empty ID should be skipped")
	}
	if _, ok := newIPAMPoolResource(types.IpamPool{
		IpamPoolId: aws.String("ipam-pool-123"),
		State:      types.IpamPoolStateCreateFailed,
	}); ok {
		t.Fatal("failed IPAM pool should be skipped")
	}
}

func TestNewIPAMPoolCIDRResource(t *testing.T) {
	resource, ok := newIPAMPoolCIDRResource("ipam-pool-123", types.IpamPoolCidr{
		Cidr:           aws.String("10.0.0.0/16"),
		IpamPoolCidrId: aws.String("ipam-pool-cidr-123"),
		State:          types.IpamPoolCidrStateProvisioned,
	})
	if !ok {
		t.Fatal("newIPAMPoolCIDRResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ipamPoolCIDRResourceType {
		t.Fatalf("resource type = %q, want %q", got, ipamPoolCIDRResourceType)
	}
	if got, want := resource.InstanceState.ID, "10.0.0.0/16_ipam-pool-123"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got := resource.InstanceState.Attributes["cidr"]; got != "10.0.0.0/16" {
		t.Fatalf("cidr = %q, want 10.0.0.0/16", got)
	}
	if got := resource.InstanceState.Attributes["ipam_pool_id"]; got != "ipam-pool-123" {
		t.Fatalf("ipam_pool_id = %q, want ipam-pool-123", got)
	}

	tests := []struct {
		name     string
		poolID   string
		poolCIDR types.IpamPoolCidr
	}{
		{name: "empty pool ID", poolCIDR: types.IpamPoolCidr{
			Cidr:  aws.String("10.0.0.0/16"),
			State: types.IpamPoolCidrStateProvisioned,
		}},
		{name: "empty cidr", poolID: "ipam-pool-123", poolCIDR: types.IpamPoolCidr{
			State: types.IpamPoolCidrStateProvisioned,
		}},
		{name: "pending provision", poolID: "ipam-pool-123", poolCIDR: types.IpamPoolCidr{
			Cidr:  aws.String("10.0.0.0/16"),
			State: types.IpamPoolCidrStatePendingProvision,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := newIPAMPoolCIDRResource(tt.poolID, tt.poolCIDR); ok {
				t.Fatal("newIPAMPoolCIDRResource() ok = true, want false")
			}
		})
	}
}

func TestIPAMPoolCIDRImportID(t *testing.T) {
	if got, want := ipamPoolCIDRImportID("10.0.0.0/16", "ipam-pool-123"), "10.0.0.0/16_ipam-pool-123"; got != want {
		t.Fatalf("ipamPoolCIDRImportID() = %q, want %q", got, want)
	}
	if got := ipamPoolCIDRImportID("", "ipam-pool-123"); got != "" {
		t.Fatalf("ipamPoolCIDRImportID(empty cidr) = %q, want empty", got)
	}
}

func TestNewIPAMResourceDiscoveryResource(t *testing.T) {
	resource, ok := newIPAMResourceDiscoveryResource(types.IpamResourceDiscovery{
		IpamResourceDiscoveryId:     aws.String("ipam-res-disco-123"),
		IpamResourceDiscoveryRegion: aws.String("us-east-1"),
		IsDefault:                   aws.Bool(false),
		State:                       types.IpamResourceDiscoveryStateCreateComplete,
	})
	if !ok {
		t.Fatal("newIPAMResourceDiscoveryResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ipamResourceDiscoveryResourceType {
		t.Fatalf("resource type = %q, want %q", got, ipamResourceDiscoveryResourceType)
	}
	if got := resource.InstanceState.ID; got != "ipam-res-disco-123" {
		t.Fatalf("resource ID = %q, want ipam-res-disco-123", got)
	}

	if _, ok := newIPAMResourceDiscoveryResource(types.IpamResourceDiscovery{
		IpamResourceDiscoveryId: aws.String("ipam-res-disco-123"),
		IsDefault:               aws.Bool(true),
		State:                   types.IpamResourceDiscoveryStateCreateComplete,
	}); ok {
		t.Fatal("default IPAM resource discovery should be skipped")
	}
	if _, ok := newIPAMResourceDiscoveryResource(types.IpamResourceDiscovery{
		IpamResourceDiscoveryId: aws.String("ipam-res-disco-123"),
		State:                   types.IpamResourceDiscoveryStateDeleteInProgress,
	}); ok {
		t.Fatal("delete-in-progress IPAM resource discovery should be skipped")
	}
}

func TestNewIPAMResourceDiscoveryAssociationResource(t *testing.T) {
	resource, ok := newIPAMResourceDiscoveryAssociationResource(types.IpamResourceDiscoveryAssociation{
		IpamId:                             aws.String("ipam-123"),
		IpamResourceDiscoveryAssociationId: aws.String("ipam-res-disco-assoc-123"),
		IpamResourceDiscoveryId:            aws.String("ipam-res-disco-123"),
		IsDefault:                          aws.Bool(false),
		ResourceDiscoveryStatus:            types.IpamAssociatedResourceDiscoveryStatusActive,
		State:                              types.IpamResourceDiscoveryAssociationStateAssociateComplete,
	})
	if !ok {
		t.Fatal("newIPAMResourceDiscoveryAssociationResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ipamResourceDiscoveryAssociationResourceType {
		t.Fatalf("resource type = %q, want %q", got, ipamResourceDiscoveryAssociationResourceType)
	}
	if got := resource.InstanceState.ID; got != "ipam-res-disco-assoc-123" {
		t.Fatalf("resource ID = %q, want ipam-res-disco-assoc-123", got)
	}

	tests := []struct {
		name        string
		association types.IpamResourceDiscoveryAssociation
	}{
		{name: "empty ID", association: types.IpamResourceDiscoveryAssociation{
			ResourceDiscoveryStatus: types.IpamAssociatedResourceDiscoveryStatusActive,
			State:                   types.IpamResourceDiscoveryAssociationStateAssociateComplete,
		}},
		{name: "default", association: types.IpamResourceDiscoveryAssociation{
			IpamResourceDiscoveryAssociationId: aws.String("ipam-res-disco-assoc-123"),
			IsDefault:                          aws.Bool(true),
			ResourceDiscoveryStatus:            types.IpamAssociatedResourceDiscoveryStatusActive,
			State:                              types.IpamResourceDiscoveryAssociationStateAssociateComplete,
		}},
		{name: "not active", association: types.IpamResourceDiscoveryAssociation{
			IpamResourceDiscoveryAssociationId: aws.String("ipam-res-disco-assoc-123"),
			ResourceDiscoveryStatus:            types.IpamAssociatedResourceDiscoveryStatusNotFound,
			State:                              types.IpamResourceDiscoveryAssociationStateAssociateComplete,
		}},
		{name: "not associated", association: types.IpamResourceDiscoveryAssociation{
			IpamResourceDiscoveryAssociationId: aws.String("ipam-res-disco-assoc-123"),
			ResourceDiscoveryStatus:            types.IpamAssociatedResourceDiscoveryStatusActive,
			State:                              types.IpamResourceDiscoveryAssociationStateDisassociateComplete,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := newIPAMResourceDiscoveryAssociationResource(tt.association); ok {
				t.Fatal("newIPAMResourceDiscoveryAssociationResource() ok = true, want false")
			}
		})
	}
}

func TestIPAMResourceNamesPreservePartBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(ipamResourceName("pool", "ab", "c"))
	right := terraformutils.TfSanitize(ipamResourceName("pool", "a", "bc"))
	if left == right {
		t.Fatalf("resource names collide: %q", left)
	}
}
