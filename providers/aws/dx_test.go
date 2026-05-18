// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/directconnect"
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
	if got, want := directConnectGatewayAssociationImportID("dxgw-123", "tgw-123"), "dxgw-123/tgw-123"; got != want {
		t.Fatalf("directConnectGatewayAssociationImportID() = %q, want %q", got, want)
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
	if resource.InstanceState.ID != "dxgw-123/tgw-123" {
		t.Fatalf("resource ID = %q, want dxgw-123/tgw-123", resource.InstanceState.ID)
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

func TestDirectConnectGatewayIDs(t *testing.T) {
	g := DirectConnectGenerator{}
	g.Resources = []terraformutils.Resource{
		terraformutils.NewSimpleResource("dxgw-123", "dxgw-123", directConnectGatewayResourceType, "aws", dxAllowEmptyValues),
		terraformutils.NewSimpleResource("", "empty-gateway", directConnectGatewayResourceType, "aws", dxAllowEmptyValues),
		terraformutils.NewSimpleResource("dxlag-123", "dxlag-123", directConnectLagResourceType, "aws", dxAllowEmptyValues),
		{},
	}

	ids := g.directConnectGatewayIDs()
	if len(ids) != 1 || ids[0] != "dxgw-123" {
		t.Fatalf("directConnectGatewayIDs() = %#v, want [dxgw-123]", ids)
	}
}

func TestDirectConnectGatewayAssociationPaginationUsesGatewayFilter(t *testing.T) {
	g := DirectConnectGenerator{}
	g.Resources = []terraformutils.Resource{
		terraformutils.NewSimpleResource("dxgw-123", "dxgw-123", directConnectGatewayResourceType, "aws", dxAllowEmptyValues),
	}
	client := &stubDirectConnectGatewayAssociationsClient{
		outputs: []*directconnect.DescribeDirectConnectGatewayAssociationsOutput{
			{
				DirectConnectGatewayAssociations: []directconnecttypes.DirectConnectGatewayAssociation{
					{
						AssociatedGateway: &directconnecttypes.AssociatedGateway{
							Id:   aws.String("tgw-123"),
							Type: directconnecttypes.GatewayTypeTransitGateway,
						},
						AssociationId:          aws.String("dxgwa-123"),
						AssociationState:       directconnecttypes.DirectConnectGatewayAssociationStateAssociated,
						DirectConnectGatewayId: aws.String("dxgw-123"),
					},
				},
				NextToken: aws.String("next"),
			},
			{
				DirectConnectGatewayAssociations: []directconnecttypes.DirectConnectGatewayAssociation{
					{
						AssociatedGateway: &directconnecttypes.AssociatedGateway{
							Id:   aws.String("tgw-dead"),
							Type: directconnecttypes.GatewayTypeTransitGateway,
						},
						AssociationId:          aws.String("dxgwa-dead"),
						AssociationState:       directconnecttypes.DirectConnectGatewayAssociationStateDisassociated,
						DirectConnectGatewayId: aws.String("dxgw-123"),
					},
				},
			},
		},
	}

	if err := g.getDirectConnectGatewayAssociations(client); err != nil {
		t.Fatalf("getDirectConnectGatewayAssociations() error = %v", err)
	}
	if len(client.requests) != 2 {
		t.Fatalf("DescribeDirectConnectGatewayAssociations calls = %d, want 2", len(client.requests))
	}
	if got := aws.ToString(client.requests[0].DirectConnectGatewayId); got != "dxgw-123" {
		t.Fatalf("first DirectConnectGatewayId = %q, want dxgw-123", got)
	}
	if client.requests[0].NextToken != nil {
		t.Fatalf("first NextToken = %q, want nil", aws.ToString(client.requests[0].NextToken))
	}
	if got := aws.ToString(client.requests[1].DirectConnectGatewayId); got != "dxgw-123" {
		t.Fatalf("second DirectConnectGatewayId = %q, want dxgw-123", got)
	}
	if got := aws.ToString(client.requests[1].NextToken); got != "next" {
		t.Fatalf("second NextToken = %q, want next", got)
	}
	if len(g.Resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(g.Resources))
	}
	association := g.Resources[1]
	if association.InstanceInfo.Type != directConnectGatewayAssociationResourceType {
		t.Fatalf("association resource type = %q, want %s", association.InstanceInfo.Type, directConnectGatewayAssociationResourceType)
	}
	if association.InstanceState.ID != "dxgw-123/tgw-123" {
		t.Fatalf("association resource ID = %q, want dxgw-123/tgw-123", association.InstanceState.ID)
	}
}

func TestDirectConnectGatewayAssociationsSkippedWithoutGateways(t *testing.T) {
	g := DirectConnectGenerator{}
	client := &stubDirectConnectGatewayAssociationsClient{}

	if err := g.getDirectConnectGatewayAssociations(client); err != nil {
		t.Fatalf("getDirectConnectGatewayAssociations() error = %v", err)
	}
	if len(client.requests) != 0 {
		t.Fatalf("DescribeDirectConnectGatewayAssociations calls = %d, want 0", len(client.requests))
	}
}

type stubDirectConnectGatewayAssociationsClient struct {
	requests []*directconnect.DescribeDirectConnectGatewayAssociationsInput
	outputs  []*directconnect.DescribeDirectConnectGatewayAssociationsOutput
}

func (c *stubDirectConnectGatewayAssociationsClient) DescribeDirectConnectGatewayAssociations(_ context.Context, input *directconnect.DescribeDirectConnectGatewayAssociationsInput, _ ...func(*directconnect.Options)) (*directconnect.DescribeDirectConnectGatewayAssociationsOutput, error) {
	inputCopy := *input
	c.requests = append(c.requests, &inputCopy)
	if len(c.requests) > len(c.outputs) {
		return &directconnect.DescribeDirectConnectGatewayAssociationsOutput{}, nil
	}
	return c.outputs[len(c.requests)-1], nil
}
