// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	directconnecttypes "github.com/aws/aws-sdk-go-v2/service/directconnect/types"
)

var dxAllowEmptyValues = []string{"tags."}

const (
	directConnectConnectionResourceType              = "aws_dx_connection"
	directConnectGatewayResourceType                 = "aws_dx_gateway"
	directConnectPrivateVirtualInterfaceResourceType = "aws_dx_private_virtual_interface"
	directConnectPublicVirtualInterfaceResourceType  = "aws_dx_public_virtual_interface"
	directConnectTransitVirtualInterfaceResourceType = "aws_dx_transit_virtual_interface"
	directConnectLagResourceType                     = "aws_dx_lag"
	directConnectGatewayAssociationResourceType      = "aws_dx_gateway_association"
)

type DirectConnectGenerator struct {
	AWSService
}

type directConnectGatewayAssociationsAPIClient interface {
	DescribeDirectConnectGatewayAssociations(context.Context, *directconnect.DescribeDirectConnectGatewayAssociationsInput, ...func(*directconnect.Options)) (*directconnect.DescribeDirectConnectGatewayAssociationsOutput, error)
}

func (g *DirectConnectGenerator) getDirectConnectGateways(svc *directconnect.Client) error {
	input := &directconnect.DescribeDirectConnectGatewaysInput{}
	for {
		output, err := svc.DescribeDirectConnectGateways(context.TODO(), input)
		if err != nil {
			return err
		}

		for _, dx := range output.DirectConnectGateways {
			if resource, ok := newDirectConnectGatewayResource(dx); ok {
				g.Resources = append(g.Resources, resource)
			}
		}

		if !awsHasMorePages(output.NextToken) {
			break
		}

		input.NextToken = output.NextToken
	}
	return nil
}

func (g *DirectConnectGenerator) getDirectConnectConnections(svc *directconnect.Client) error {
	input := &directconnect.DescribeConnectionsInput{}
	output, err := svc.DescribeConnections(context.TODO(), input)
	if err != nil {
		return err
	}

	for _, dx := range output.Connections {
		if resource, ok := newDirectConnectConnectionResource(dx); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *DirectConnectGenerator) getDirectConnectVirtualInterfaces(svc *directconnect.Client, currentAccountID string) error {
	input := &directconnect.DescribeVirtualInterfacesInput{}
	output, err := svc.DescribeVirtualInterfaces(context.TODO(), input)
	if err != nil {
		return err
	}

	for _, vif := range output.VirtualInterfaces {
		resourceType, ok := directConnectVirtualInterfaceResourceType(vif)
		if !ok {
			log.Printf("Unknown Virtual Interface Type: %s for ID: %s", StringValue(vif.VirtualInterfaceType), StringValue(vif.VirtualInterfaceId))
			continue
		}

		if resource, ok := newDirectConnectVirtualInterfaceResource(vif, resourceType, currentAccountID); ok {
			g.Resources = append(g.Resources, resource)
		}
	}

	return nil
}

func directConnectVirtualInterfaceResourceType(vif directconnecttypes.VirtualInterface) (string, bool) {
	switch StringValue(vif.VirtualInterfaceType) {
	case "private":
		return directConnectPrivateVirtualInterfaceResourceType, true
	case "public":
		return directConnectPublicVirtualInterfaceResourceType, true
	case "transit":
		return directConnectTransitVirtualInterfaceResourceType, true
	default:
		return "", false
	}
}

func (g *DirectConnectGenerator) getDirectConnectLags(svc *directconnect.Client) error {
	input := &directconnect.DescribeLagsInput{}
	for {
		output, err := svc.DescribeLags(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, lag := range output.Lags {
			if resource, ok := newDirectConnectLagResource(lag); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return nil
}

func (g *DirectConnectGenerator) getDirectConnectGatewayAssociations(svc directConnectGatewayAssociationsAPIClient, region string) error {
	for _, gatewayID := range g.directConnectGatewayIDs() {
		if err := g.getDirectConnectGatewayAssociationsForGateway(svc, gatewayID, region); err != nil {
			return err
		}
	}
	return nil
}

func (g *DirectConnectGenerator) directConnectGatewayIDs() []string {
	ids := []string{}
	for _, resource := range g.Resources {
		if resource.InstanceInfo == nil || resource.InstanceState == nil {
			continue
		}
		if resource.InstanceInfo.Type != directConnectGatewayResourceType || resource.InstanceState.ID == "" {
			continue
		}
		ids = append(ids, resource.InstanceState.ID)
	}
	return ids
}

func (g *DirectConnectGenerator) getDirectConnectGatewayAssociationsForGateway(svc directConnectGatewayAssociationsAPIClient, gatewayID, region string) error {
	if gatewayID == "" {
		return nil
	}
	input := &directconnect.DescribeDirectConnectGatewayAssociationsInput{
		DirectConnectGatewayId: aws.String(gatewayID),
	}
	for {
		output, err := svc.DescribeDirectConnectGatewayAssociations(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, association := range output.DirectConnectGatewayAssociations {
			if !directConnectGatewayAssociationRegionMatches(association, region) {
				continue
			}
			if resource, ok := newDirectConnectGatewayAssociationResource(association); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return nil
}

func newDirectConnectConnectionResource(connection directconnecttypes.Connection) (terraformutils.Resource, bool) {
	if !directConnectConnectionImportable(connection) || StringValue(connection.ConnectionId) == "" {
		return terraformutils.Resource{}, false
	}
	resourceName := StringValue(connection.ConnectionName)
	if resourceName == "" {
		resourceName = StringValue(connection.ConnectionId)
	}
	return terraformutils.NewSimpleResource(
		StringValue(connection.ConnectionId),
		resourceName,
		directConnectConnectionResourceType,
		"aws",
		dxAllowEmptyValues,
	), true
}

func newDirectConnectGatewayResource(gateway directconnecttypes.DirectConnectGateway) (terraformutils.Resource, bool) {
	gatewayID := StringValue(gateway.DirectConnectGatewayId)
	if gatewayID == "" || !directConnectGatewayImportable(gateway) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		gatewayID,
		gatewayID,
		directConnectGatewayResourceType,
		"aws",
		dxAllowEmptyValues,
	), true
}

func newDirectConnectVirtualInterfaceResource(vif directconnecttypes.VirtualInterface, resourceType, currentAccountID string) (terraformutils.Resource, bool) {
	if !directConnectVirtualInterfaceImportable(vif, currentAccountID) || StringValue(vif.VirtualInterfaceId) == "" || resourceType == "" {
		return terraformutils.Resource{}, false
	}
	resourceName := StringValue(vif.VirtualInterfaceName)
	if resourceName == "" {
		resourceName = StringValue(vif.VirtualInterfaceId)
	}
	return terraformutils.NewSimpleResource(
		StringValue(vif.VirtualInterfaceId),
		resourceName,
		resourceType,
		"aws",
		dxAllowEmptyValues,
	), true
}

func newDirectConnectLagResource(lag directconnecttypes.Lag) (terraformutils.Resource, bool) {
	if !directConnectLagImportable(lag) || StringValue(lag.LagId) == "" {
		return terraformutils.Resource{}, false
	}
	resourceName := StringValue(lag.LagName)
	if resourceName == "" {
		resourceName = StringValue(lag.LagId)
	}
	return terraformutils.NewSimpleResource(
		StringValue(lag.LagId),
		resourceName,
		directConnectLagResourceType,
		"aws",
		dxAllowEmptyValues,
	), true
}

func newDirectConnectGatewayAssociationResource(association directconnecttypes.DirectConnectGatewayAssociation) (terraformutils.Resource, bool) {
	if !directConnectGatewayAssociationImportable(association) {
		return terraformutils.Resource{}, false
	}
	dxGatewayID := StringValue(association.DirectConnectGatewayId)
	associationID := StringValue(association.AssociationId)
	associatedGatewayID := ""
	associatedGatewayType := ""
	if association.AssociatedGateway != nil {
		associatedGatewayID = StringValue(association.AssociatedGateway.Id)
		associatedGatewayType = string(association.AssociatedGateway.Type)
	}
	if dxGatewayID == "" || associationID == "" || associatedGatewayID == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		directConnectGatewayAssociationStateID(dxGatewayID, associatedGatewayID),
		awsResourceNameWithLengths("gateway_association", dxGatewayID, associatedGatewayID),
		directConnectGatewayAssociationResourceType,
		"aws",
		map[string]string{
			"associated_gateway_id":       associatedGatewayID,
			"associated_gateway_type":     associatedGatewayType,
			"dx_gateway_association_id":   associationID,
			"dx_gateway_id":               dxGatewayID,
			"dx_gateway_owner_account_id": StringValue(association.DirectConnectGatewayOwnerAccount),
		},
		dxAllowEmptyValues,
		map[string]interface{}{},
	)
	setDirectConnectImportID(&resource, directConnectGatewayAssociationImportID(dxGatewayID, associatedGatewayID))
	resource.IgnoreKeys = append(resource.IgnoreKeys, "^associated_gateway_owner_account_id$")
	return resource, true
}

func directConnectConnectionImportable(connection directconnecttypes.Connection) bool {
	if StringValue(connection.LagId) != "" {
		return false
	}
	switch connection.ConnectionState {
	case directconnecttypes.ConnectionStateDeleting,
		directconnecttypes.ConnectionStateDeleted,
		directconnecttypes.ConnectionStateRejected:
		return false
	default:
		return true
	}
}

func directConnectGatewayImportable(gateway directconnecttypes.DirectConnectGateway) bool {
	switch gateway.DirectConnectGatewayState {
	case directconnecttypes.DirectConnectGatewayStateDeleting,
		directconnecttypes.DirectConnectGatewayStateDeleted:
		return false
	default:
		return true
	}
}

func directConnectVirtualInterfaceImportable(vif directconnecttypes.VirtualInterface, currentAccountID string) bool {
	switch vif.VirtualInterfaceState {
	case directconnecttypes.VirtualInterfaceStateDeleting,
		directconnecttypes.VirtualInterfaceStateDeleted,
		directconnecttypes.VirtualInterfaceStateConfirming,
		directconnecttypes.VirtualInterfaceStateRejected:
		return false
	}
	return !directConnectHostedTransitVirtualInterface(vif, currentAccountID)
}

func directConnectHostedTransitVirtualInterface(vif directconnecttypes.VirtualInterface, currentAccountID string) bool {
	ownerAccountID := StringValue(vif.OwnerAccount)
	return currentAccountID != "" &&
		ownerAccountID != "" &&
		ownerAccountID != currentAccountID &&
		StringValue(vif.VirtualInterfaceType) == "transit"
}

func directConnectLagImportable(lag directconnecttypes.Lag) bool {
	switch lag.LagState {
	case directconnecttypes.LagStateDeleting,
		directconnecttypes.LagStateDeleted:
		return false
	default:
		return true
	}
}

func directConnectGatewayAssociationImportable(association directconnecttypes.DirectConnectGatewayAssociation) bool {
	switch association.AssociationState {
	case directconnecttypes.DirectConnectGatewayAssociationStateDisassociating,
		directconnecttypes.DirectConnectGatewayAssociationStateDisassociated:
		return false
	}
	return !directConnectGatewayAssociationRequiresProposal(association)
}

func directConnectGatewayAssociationRequiresProposal(association directconnecttypes.DirectConnectGatewayAssociation) bool {
	if association.AssociatedGateway == nil {
		return false
	}
	associatedGatewayOwnerAccountID := StringValue(association.AssociatedGateway.OwnerAccount)
	dxGatewayOwnerAccountID := StringValue(association.DirectConnectGatewayOwnerAccount)
	return associatedGatewayOwnerAccountID != "" &&
		dxGatewayOwnerAccountID != "" &&
		associatedGatewayOwnerAccountID != dxGatewayOwnerAccountID
}

func directConnectGatewayAssociationRegionMatches(association directconnecttypes.DirectConnectGatewayAssociation, region string) bool {
	if association.AssociatedGateway == nil || region == "" {
		return true
	}
	associatedRegion := StringValue(association.AssociatedGateway.Region)
	return associatedRegion == "" || associatedRegion == region
}

func directConnectGatewayAssociationImportID(dxGatewayID, associatedGatewayID string) string {
	return dxGatewayID + "/" + associatedGatewayID
}

func directConnectGatewayAssociationStateID(dxGatewayID, associatedGatewayID string) string {
	return "ga-" + dxGatewayID + associatedGatewayID
}

func setDirectConnectImportID(resource *terraformutils.Resource, importID string) {
	if resource == nil || resource.InstanceState == nil || importID == "" {
		return
	}
	if resource.InstanceState.Meta == nil {
		resource.InstanceState.Meta = map[string]interface{}{}
	}
	resource.InstanceState.Meta["import_id"] = importID
}

func (g *DirectConnectGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := directconnect.NewFromConfig(config)
	currentAccountID := ""
	if account, err := g.getAccountNumber(config); err != nil {
		log.Printf("Skipping Direct Connect hosted transit VIF owner filtering: unable to get account ID: %v", err)
	} else {
		currentAccountID = StringValue(account)
	}
	if err := g.getDirectConnectGateways(svc); err != nil {
		return err
	}

	err = g.getDirectConnectVirtualInterfaces(svc, currentAccountID)
	if err != nil {
		return err
	}

	err = g.getDirectConnectLags(svc)
	if err != nil {
		return err
	}

	err = g.getDirectConnectGatewayAssociations(svc, config.Region)
	if err != nil {
		return err
	}

	err = g.getDirectConnectConnections(svc)
	if err != nil {
		return err
	}

	return nil
}
