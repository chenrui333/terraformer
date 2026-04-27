// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// VPNGatewayGenerator ...
type VPNGatewayGenerator struct {
	IBMService
}

func (g VPNGatewayGenerator) createVPNGatewayResources(vpngwID, vpngwName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		vpngwID,
		normalizeResourceName(vpngwName, false),
		"ibm_is_vpn_gateway",
		"ibm",
		[]string{})
	return resources
}

func (g VPNGatewayGenerator) createVPNGatewayConnectionResources(vpngwID, vpngwConnectionID, vpngwConnectionName string) terraformutils.Resource {
	resources := terraformutils.NewResource(
		fmt.Sprintf("%s/%s", vpngwID, vpngwConnectionID),
		normalizeResourceName(vpngwConnectionName, false),
		"ibm_is_vpn_gateway_connection",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{})
	return resources
}

// InitResources ...
func (g *VPNGatewayGenerator) InitResources() error {
	region := g.Args["region"].(string)
	apiKey := os.Getenv("IC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("no API key set")
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
	var allrecs []vpcv1.VPNGatewayIntf
	for {
		listVPNGatewaysOptions := &vpcv1.ListVPNGatewaysOptions{}
		if start != "" {
			listVPNGatewaysOptions.Start = &start
		}
		if rg := g.Args["resource_group"].(string); rg != "" {
			rg, err = GetResourceGroupID(apiKey, rg, region)
			if err != nil {
				return fmt.Errorf("error fetching Resource Group Id %w", err)
			}
			listVPNGatewaysOptions.ResourceGroupID = &rg
		}
		vpngws, response, err := vpcclient.ListVPNGateways(listVPNGatewaysOptions)
		if err != nil {
			return fmt.Errorf("error fetching VPN Gateways %w\n%s", err, response)
		}
		start = GetNext(vpngws.Next)
		allrecs = append(allrecs, vpngws.VPNGateways...)
		if start == "" {
			break
		}
	}

	for _, gw := range allrecs {
		vpngw := gw.(*vpcv1.VPNGateway)
		g.Resources = append(g.Resources, g.createVPNGatewayResources(*vpngw.ID, *vpngw.Name))
		listVPNGatewayConnectionsOptions := &vpcv1.ListVPNGatewayConnectionsOptions{
			VPNGatewayID: vpngw.ID,
		}
		vpngwConnections, response, err := vpcclient.ListVPNGatewayConnections(listVPNGatewayConnectionsOptions)
		if err != nil {
			return fmt.Errorf("error fetching VPN Gateway Connections %w\n%s", err, response)
		}
		for _, connection := range vpngwConnections.Connections {
			vpngwConnection := connection.(*vpcv1.VPNGatewayConnection)
			g.Resources = append(g.Resources, g.createVPNGatewayConnectionResources(*vpngw.ID, *vpngwConnection.ID, *vpngwConnection.Name))
		}
	}
	return nil
}

func (g *VPNGatewayGenerator) PostConvertHook() error {
	for i, con := range g.Resources {
		if con.InstanceInfo.Type != "ibm_is_vpn_gateway_connection" {
			continue
		}
		for _, vpn := range g.Resources {
			if vpn.InstanceInfo.Type != "ibm_is_vpn_gateway" {
				continue
			}
			if con.InstanceState.Attributes["vpn_gateway"] == vpn.InstanceState.Attributes["id"] {
				g.Resources[i].Item["vpn_gateway"] = "${ibm_is_vpn_gateway." + vpn.ResourceName + ".id}"
			}
		}
	}

	return nil
}
