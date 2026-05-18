// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	directconnecttypes "github.com/aws/aws-sdk-go-v2/service/directconnect/types"
)

func TestDirectConnectVirtualInterfaceResources(t *testing.T) {
	resource, ok := newDirectConnectVirtualInterfaceResource(directconnecttypes.VirtualInterface{
		VirtualInterfaceId:    aws.String("dxvif-transit"),
		VirtualInterfaceName:  aws.String("transit-core"),
		VirtualInterfaceState: directconnecttypes.VirtualInterfaceStateAvailable,
		VirtualInterfaceType:  aws.String("transit"),
	}, directConnectTransitVirtualInterfaceResourceType)
	if !ok {
		t.Fatal("expected transit virtual interface resource")
	}
	if resource.InstanceState.ID != "dxvif-transit" {
		t.Fatalf("resource ID = %q, want dxvif-transit", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != directConnectTransitVirtualInterfaceResourceType {
		t.Fatalf("resource type = %q, want %s", resource.InstanceInfo.Type, directConnectTransitVirtualInterfaceResourceType)
	}

	if _, ok := newDirectConnectVirtualInterfaceResource(directconnecttypes.VirtualInterface{
		VirtualInterfaceId:    aws.String("dxvif-deleted"),
		VirtualInterfaceState: directconnecttypes.VirtualInterfaceStateDeleted,
	}, directConnectTransitVirtualInterfaceResourceType); ok {
		t.Fatal("deleted virtual interface should be skipped")
	}
}

func TestDirectConnectLagResourceFiltersTerminalStates(t *testing.T) {
	resource, ok := newDirectConnectLagResource(directconnecttypes.Lag{
		LagId:    aws.String("dxlag-123"),
		LagName:  aws.String("core-lag"),
		LagState: directconnecttypes.LagStateAvailable,
	})
	if !ok {
		t.Fatal("expected LAG resource")
	}
	if resource.InstanceState.ID != "dxlag-123" {
		t.Fatalf("resource ID = %q, want dxlag-123", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != directConnectLagResourceType {
		t.Fatalf("resource type = %q, want %s", resource.InstanceInfo.Type, directConnectLagResourceType)
	}

	if _, ok := newDirectConnectLagResource(directconnecttypes.Lag{
		LagId:    aws.String("dxlag-deleting"),
		LagState: directconnecttypes.LagStateDeleting,
	}); ok {
		t.Fatal("deleting LAG should be skipped")
	}
}

func TestDirectConnectGatewayAssociationResource(t *testing.T) {
	if got, want := directConnectGatewayAssociationStateID("dxgw-123", "tgw-123"), "ga-dxgw-123tgw-123"; got != want {
		t.Fatalf("directConnectGatewayAssociationStateID() = %q, want %q", got, want)
	}

	resource, ok := newDirectConnectGatewayAssociationResource(directconnecttypes.DirectConnectGatewayAssociation{
		AssociatedGateway: &directconnecttypes.AssociatedGateway{
			Id:           aws.String("tgw-123"),
			OwnerAccount: aws.String("123456789012"),
			Type:         directconnecttypes.GatewayTypeTransitGateway,
		},
		AssociationId:                    aws.String("dxgwa-123"),
		AssociationState:                 directconnecttypes.DirectConnectGatewayAssociationStateAssociated,
		DirectConnectGatewayId:           aws.String("dxgw-123"),
		DirectConnectGatewayOwnerAccount: aws.String("210987654321"),
	})
	if !ok {
		t.Fatal("expected gateway association resource")
	}
	if resource.InstanceState.ID != "ga-dxgw-123tgw-123" {
		t.Fatalf("resource ID = %q, want ga-dxgw-123tgw-123", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != directConnectGatewayAssociationResourceType {
		t.Fatalf("resource type = %q, want %s", resource.InstanceInfo.Type, directConnectGatewayAssociationResourceType)
	}
	if got := resource.InstanceState.Attributes["dx_gateway_association_id"]; got != "dxgwa-123" {
		t.Fatalf("dx_gateway_association_id = %q, want dxgwa-123", got)
	}

	if _, ok := newDirectConnectGatewayAssociationResource(directconnecttypes.DirectConnectGatewayAssociation{
		AssociationId:          aws.String("dxgwa-dead"),
		AssociationState:       directconnecttypes.DirectConnectGatewayAssociationStateDisassociated,
		DirectConnectGatewayId: aws.String("dxgw-123"),
	}); ok {
		t.Fatal("disassociated gateway association should be skipped")
	}
}

func TestDirectConnectConnectionResourceFiltersTerminalStates(t *testing.T) {
	if _, ok := newDirectConnectConnectionResource(directconnecttypes.Connection{
		ConnectionId:    aws.String("dxcon-rejected"),
		ConnectionState: directconnecttypes.ConnectionStateRejected,
	}); ok {
		t.Fatal("rejected connection should be skipped")
	}
}
