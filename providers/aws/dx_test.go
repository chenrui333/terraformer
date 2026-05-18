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
	currentAccountID := "123456789012"
	resource, ok := newDirectConnectVirtualInterfaceResource(directconnecttypes.VirtualInterface{
		VirtualInterfaceId:    aws.String("dxvif-transit"),
		VirtualInterfaceName:  aws.String("transit-core"),
		OwnerAccount:          aws.String(currentAccountID),
		VirtualInterfaceState: directconnecttypes.VirtualInterfaceStateAvailable,
		VirtualInterfaceType:  aws.String("transit"),
	}, directConnectTransitVirtualInterfaceResourceType, currentAccountID)
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
	}, directConnectTransitVirtualInterfaceResourceType, currentAccountID); ok {
		t.Fatal("deleted virtual interface should be skipped")
	}
	if _, ok := newDirectConnectVirtualInterfaceResource(directconnecttypes.VirtualInterface{
		VirtualInterfaceId:    aws.String("dxvif-confirming"),
		VirtualInterfaceState: directconnecttypes.VirtualInterfaceStateConfirming,
		VirtualInterfaceType:  aws.String("transit"),
	}, directConnectTransitVirtualInterfaceResourceType, currentAccountID); ok {
		t.Fatal("confirming hosted transit virtual interface should be skipped")
	}
	if _, ok := newDirectConnectVirtualInterfaceResource(directconnecttypes.VirtualInterface{
		VirtualInterfaceId:    aws.String("dxvif-hosted-transit"),
		OwnerAccount:          aws.String("210987654321"),
		VirtualInterfaceState: directconnecttypes.VirtualInterfaceStateAvailable,
		VirtualInterfaceType:  aws.String("transit"),
	}, directConnectTransitVirtualInterfaceResourceType, currentAccountID); ok {
		t.Fatal("accepted hosted transit virtual interface should be skipped")
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
	if got := resource.InstanceState.Meta["import_id"]; got != "dxgw-123/tgw-123" {
		t.Fatalf("import_id = %#v, want dxgw-123/tgw-123", got)
	}
	if resource.InstanceInfo.Type != directConnectGatewayAssociationResourceType {
		t.Fatalf("resource type = %q, want %s", resource.InstanceInfo.Type, directConnectGatewayAssociationResourceType)
	}
	if got := resource.InstanceState.Attributes["dx_gateway_association_id"]; got != "dxgwa-123" {
		t.Fatalf("dx_gateway_association_id = %q, want dxgwa-123", got)
	}
	if _, ok := resource.InstanceState.Attributes["associated_gateway_owner_account_id"]; ok {
		t.Fatal("associated_gateway_owner_account_id should not be seeded with associated_gateway_id")
	}
	if !directConnectTestStringSliceContains(resource.IgnoreKeys, "^associated_gateway_owner_account_id$") {
		t.Fatalf("IgnoreKeys = %#v, want associated_gateway_owner_account_id ignored", resource.IgnoreKeys)
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
	if _, ok := newDirectConnectConnectionResource(directconnecttypes.Connection{
		ConnectionId:    aws.String("dxcon-lag-member"),
		ConnectionState: directconnecttypes.ConnectionStateAvailable,
		LagId:           aws.String("dxlag-123"),
	}); ok {
		t.Fatal("LAG member connection should be skipped")
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

	if err := g.getDirectConnectGatewayAssociations(client, "us-east-1"); err != nil {
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
	if association.InstanceState.ID != "ga-dxgw-123tgw-123" {
		t.Fatalf("association resource ID = %q, want ga-dxgw-123tgw-123", association.InstanceState.ID)
	}
	if got := association.InstanceState.Meta["import_id"]; got != "dxgw-123/tgw-123" {
		t.Fatalf("association import_id = %#v, want dxgw-123/tgw-123", got)
	}
}

func TestDirectConnectGatewayAssociationsSkipMismatchedRegion(t *testing.T) {
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
							Id:     aws.String("tgw-west"),
							Region: aws.String("us-west-2"),
							Type:   directconnecttypes.GatewayTypeTransitGateway,
						},
						AssociationId:          aws.String("dxgwa-west"),
						AssociationState:       directconnecttypes.DirectConnectGatewayAssociationStateAssociated,
						DirectConnectGatewayId: aws.String("dxgw-123"),
					},
					{
						AssociatedGateway: &directconnecttypes.AssociatedGateway{
							Id:     aws.String("tgw-east"),
							Region: aws.String("us-east-1"),
							Type:   directconnecttypes.GatewayTypeTransitGateway,
						},
						AssociationId:          aws.String("dxgwa-east"),
						AssociationState:       directconnecttypes.DirectConnectGatewayAssociationStateAssociated,
						DirectConnectGatewayId: aws.String("dxgw-123"),
					},
				},
			},
		},
	}

	if err := g.getDirectConnectGatewayAssociations(client, "us-east-1"); err != nil {
		t.Fatalf("getDirectConnectGatewayAssociations() error = %v", err)
	}
	if len(g.Resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(g.Resources))
	}
	association := g.Resources[1]
	if association.InstanceState.ID != "ga-dxgw-123tgw-east" {
		t.Fatalf("association resource ID = %q, want ga-dxgw-123tgw-east", association.InstanceState.ID)
	}
	if got := association.InstanceState.Meta["import_id"]; got != "dxgw-123/tgw-east" {
		t.Fatalf("association import_id = %#v, want dxgw-123/tgw-east", got)
	}
}

func TestDirectConnectGatewayAssociationsSkippedWithoutGateways(t *testing.T) {
	g := DirectConnectGenerator{}
	client := &stubDirectConnectGatewayAssociationsClient{}

	if err := g.getDirectConnectGatewayAssociations(client, "us-east-1"); err != nil {
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

func directConnectTestStringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
