// Copyright 2019 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	dlProviderV2 "github.com/IBM/networking-go-sdk/directlinkproviderv2"
	dl "github.com/IBM/networking-go-sdk/directlinkv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// DLGenerator ...
type DLGenerator struct {
	IBMService
}

func (g DLGenerator) createDirectLinkGatewayResources(gatewayID, gatewayName string) terraformutils.Resource {
	resource := terraformutils.NewSimpleResource(
		gatewayID,
		normalizeResourceName(gatewayName, false),
		"ibm_dl_gateway",
		"ibm",
		[]string{})
	return resource
}

func (g DLGenerator) createDirectLinkVirtualConnectionResources(gatewayID, connectionID, connectionName string, dependsOn []string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		fmt.Sprintf("%s/%s", gatewayID, connectionID),
		normalizeResourceName(connectionName, false),
		"ibm_dl_virtual_connection",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{
			"depends_on": dependsOn,
		})
	return resource
}

func (g DLGenerator) createDirectLinkProviderGatewayResources(providerGatewayID, providerGatewayName string) terraformutils.Resource {
	resource := terraformutils.NewSimpleResource(
		providerGatewayID,
		normalizeResourceName(providerGatewayName, false),
		"ibm_dl_provider_gateway",
		"ibm",
		[]string{})
	return resource
}

// InitResources ...
func (g *DLGenerator) InitResources() error {
	apiKey := os.Getenv("IC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("no API key set")
	}
	dlURL := "https://directlink.cloud.ibm.com/v1"
	directlinkOptions := &dl.DirectLinkV1Options{
		URL: envFallBack([]string{"IBMCLOUD_DL_API_ENDPOINT"}, dlURL),
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
		},
		Version: CreateVersionDate(),
	}
	dlclient, err := dl.NewDirectLinkV1(directlinkOptions)
	if err != nil {
		return err
	}

	listGatewaysOptions := &dl.ListGatewaysOptions{}
	gateways, response, err := dlclient.ListGateways(listGatewaysOptions)
	if err != nil {
		return fmt.Errorf("error fetching Direct Link Gateways %w\n%s", err, response)
	}
	if gateways.Gateways != nil {
		for _, gateway := range gateways.Gateways {
			gatewayID, gatewayName, err := directLinkGatewayIDName(gateway)
			if err != nil {
				return err
			}

			g.Resources = append(g.Resources, g.createDirectLinkGatewayResources(*gatewayID, *gatewayName))
			resourceName := g.Resources[len(g.Resources)-1:][0].ResourceName
			var dependsOn []string
			dependsOn = append(dependsOn, "ibm_dl_gateway."+resourceName)
			listGatewayVirtualConnectionsOptions := &dl.ListGatewayVirtualConnectionsOptions{
				GatewayID: gatewayID,
			}
			connections, response, err := dlclient.ListGatewayVirtualConnections(listGatewayVirtualConnectionsOptions)
			if err != nil {
				return fmt.Errorf("error fetching Direct Link Virtual connections %w\n%s", err, response)
			}
			for _, connection := range connections.VirtualConnections {
				g.Resources = append(g.Resources, g.createDirectLinkVirtualConnectionResources(*gatewayID, *connection.ID, *connection.Name, dependsOn))
			}
		}
	}

	dlproviderURL := "https://directlink.cloud.ibm.com/provider/v2"
	dlproviderOptions := &dlProviderV2.DirectLinkProviderV2Options{
		URL: envFallBack([]string{"IBMCLOUD_DL_PROVIDER_API_ENDPOINT"}, dlproviderURL),
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
		},
		Version: CreateVersionDate(),
	}
	dlproviderclient, err := dlProviderV2.NewDirectLinkProviderV2(dlproviderOptions)
	if err != nil {
		return err
	}
	start := ""
	allrecs := []dlProviderV2.ProviderGateway{}
	for {
		listProviderGatewaysOptions := &dlProviderV2.ListProviderGatewaysOptions{}
		if start != "" {
			listProviderGatewaysOptions.Start = &start
		}

		providerGateways, resp, err := dlproviderclient.ListProviderGateways(listProviderGatewaysOptions)
		if err != nil {
			return fmt.Errorf("error listing Direct Link Provider Gateways %w\n%s", err, resp)
		}
		start = GetNext(providerGateways.Next)
		allrecs = append(allrecs, providerGateways.Gateways...)
		if start == "" {
			break
		}
	}
	for _, providerGateway := range allrecs {
		g.Resources = append(g.Resources, g.createDirectLinkProviderGatewayResources(*providerGateway.ID, *providerGateway.Name))
	}
	return nil
}

func directLinkGatewayIDName(gateway dl.GatewayCollectionGatewaysItemIntf) (*string, *string, error) {
	var id *string
	var name *string

	switch gateway := gateway.(type) {
	case *dl.GatewayCollectionGatewaysItem:
		id = gateway.ID
		name = gateway.Name
	case *dl.GatewayCollectionGatewaysItemGateway:
		id = gateway.ID
		name = gateway.Name
	case *dl.GatewayCollectionGatewaysItemCrossAccountGateway:
		id = gateway.ID
		name = gateway.Name
	default:
		return nil, nil, fmt.Errorf("unsupported Direct Link gateway type %T", gateway)
	}

	if id == nil || name == nil {
		return nil, nil, fmt.Errorf("direct link gateway missing id or name")
	}

	return id, name, nil
}
