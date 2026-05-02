// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
)

var dxAllowEmptyValues = []string{"tags."}

type DirectConnectGenerator struct {
	AWSService
}

func (g *DirectConnectGenerator) getDirectConnectGateways(svc *directconnect.Client) error {
	input := &directconnect.DescribeDirectConnectGatewaysInput{}
	for {
		// Fetch a page of results
		output, err := svc.DescribeDirectConnectGateways(context.TODO(), input)
		if err != nil {
			return err
		}

		// Process each DirectConnect Gateway
		for _, dx := range output.DirectConnectGateways {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*dx.DirectConnectGatewayId, // Dereference the pointer
				*dx.DirectConnectGatewayId,
				"aws_dx_gateway",
				"aws",
				dxAllowEmptyValues,
			))
		}

		// Check if there are more pages
		if !awsHasMorePages(output.NextToken) {
			break
		}

		// Update the input token for the next page
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
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			*dx.ConnectionId, // Dereference the pointer
			*dx.ConnectionName,
			"aws_dx_connection",
			"aws",
			dxAllowEmptyValues,
		))
	}
	return nil
}

func (g *DirectConnectGenerator) getDirectConnectVritualInterfaces(svc *directconnect.Client) error {
	input := &directconnect.DescribeVirtualInterfacesInput{}
	output, err := svc.DescribeVirtualInterfaces(context.TODO(), input)
	if err != nil {
		return err
	}

	for _, vif := range output.VirtualInterfaces {
		var resourceType string

		switch *vif.VirtualInterfaceType {
		case "private":
			resourceType = "aws_dx_private_virtual_interface"
		case "public":
			resourceType = "aws_dx_public_virtual_interface"
		default:
			log.Printf("Unknown Virtual Interface Type: %s for ID: %s", *vif.VirtualInterfaceType, *vif.VirtualInterfaceId)
			continue
		}

		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			*vif.VirtualInterfaceId,
			*vif.VirtualInterfaceName,
			resourceType,
			"aws",
			dxAllowEmptyValues,
		))
	}

	return nil
}

func (g *DirectConnectGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := directconnect.NewFromConfig(config)
	if err := g.getDirectConnectGateways(svc); err != nil {
		return err
	}

	err = g.getDirectConnectVritualInterfaces(svc)
	if err != nil {
		return err
	}

	err = g.getDirectConnectConnections(svc)
	if err != nil {
		return err
	}

	return nil
}
