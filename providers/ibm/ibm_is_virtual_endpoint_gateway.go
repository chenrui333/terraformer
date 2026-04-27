// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// VPEGenerator ...
type VPEGenerator struct {
	IBMService
}

func (g VPEGenerator) createVPEGatewayResources(gatewayID, gatewayName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		gatewayID,
		normalizeResourceName(gatewayName, false),
		"ibm_is_virtual_endpoint_gateway",
		"ibm",
		[]string{})
	return resources
}

func (g VPEGenerator) createVPEGatewayIPResources(gatewayID, gatewayIPID, gatewayIPName string) terraformutils.Resource {
	resources := terraformutils.NewResource(
		fmt.Sprintf("%s/%s", gatewayID, gatewayIPID),
		normalizeResourceName(gatewayIPName, false),
		"ibm_is_virtual_endpoint_gateway_ip",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{})
	return resources
}

// InitResources ...
func (g *VPEGenerator) InitResources() error {
	region := g.Args["region"].(string)
	apiKey := os.Getenv("IC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("No API key set")
	}

	isURL := GetVPCEndPoint(region)
	iamURL := GetAuthEndPoint()
	vpcoptions := &vpcv1.VpcV1Options{
		URL: isURL,
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
			URL:    iamURL,
		},
	}
	vpcclient, err := vpcv1.NewVpcV1(vpcoptions)
	if err != nil {
		return err
	}

	start := ""
	allrecs := []vpcv1.EndpointGateway{}
	for {
		listEndpointGatewaysOptions := &vpcv1.ListEndpointGatewaysOptions{}
		if start != "" {
			listEndpointGatewaysOptions.Start = &start
		}
		if rg := g.Args["resource_group"].(string); rg != "" {
			rg, err = GetResourceGroupID(apiKey, rg, region)
			if err != nil {
				return fmt.Errorf("error fetching Resource Group Id %w", err)
			}
			listEndpointGatewaysOptions.ResourceGroupID = &rg
		}
		gateways, response, err := vpcclient.ListEndpointGateways(listEndpointGatewaysOptions)
		if err != nil {
			return fmt.Errorf("error fetching endpoint gateways %w\n%s", err, response)
		}
		start = GetNext(gateways.Next)
		allrecs = append(allrecs, gateways.EndpointGateways...)
		if start == "" {
			break
		}
	}

	for _, gateway := range allrecs {
		start := ""
		allrecs := []vpcv1.ReservedIP{}
		g.Resources = append(g.Resources, g.createVPEGatewayResources(*gateway.ID, *gateway.Name))
		listEndpointGatewayIpsOptions := &vpcv1.ListEndpointGatewayIpsOptions{
			EndpointGatewayID: gateway.ID,
		}
		if start != "" {
			listEndpointGatewayIpsOptions.Start = &start
		}
		ips, response, err := vpcclient.ListEndpointGatewayIps(listEndpointGatewayIpsOptions)
		if err != nil {
			return fmt.Errorf("error fetching endpoint gateway ips %w\n%s", err, response)
		}
		start = GetNext(ips.Next)
		allrecs = append(allrecs, ips.Ips...)
		if start == "" {
			break
		}
		for _, ip := range allrecs {
			g.Resources = append(g.Resources, g.createVPEGatewayIPResources(*gateway.ID, *ip.ID, *ip.Name))
		}
	}
	return nil
}

func (g *VPEGenerator) PostConvertHook() error {
	for i, r := range g.Resources {
		if r.InstanceInfo.Type != "ibm_is_virtual_endpoint_gateway" {
			continue
		}
		for _, gIP := range g.Resources {
			if gIP.InstanceInfo.Type != "ibm_is_virtual_endpoint_gateway_ip" {
				continue
			}
			if gIP.InstanceState.Attributes["gateway"] == r.InstanceState.Attributes["id"] {
				g.Resources[i].Item["gateway"] = "${ibm_is_virtual_endpoint_gateway." + r.ResourceName + ".id}"
			}
		}
	}

	return nil
}
