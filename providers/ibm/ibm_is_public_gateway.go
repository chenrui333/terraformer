// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// PublicGatewayGenerator ...
type PublicGatewayGenerator struct {
	IBMService
}

func (g PublicGatewayGenerator) createPublicGatewayResources(publicGatewayID, publicGatewayName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		publicGatewayID,
		normalizeResourceName(publicGatewayName, false),
		"ibm_is_public_gateway",
		"ibm",
		[]string{})
	return resources
}

// InitResources ...
func (g *PublicGatewayGenerator) InitResources() error {
	region := g.Args["region"].(string)
	apiKey := os.Getenv("IC_API_KEY")
	if apiKey == "" {
		return errMissingICAPIKey
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
	var allrecs []vpcv1.PublicGateway
	for {
		options := &vpcv1.ListPublicGatewaysOptions{}
		if start != "" {
			options.Start = &start
		}
		if rg := g.Args["resource_group"].(string); rg != "" {
			rg, err = GetResourceGroupID(apiKey, rg, region)
			if err != nil {
				return fmt.Errorf("error fetching Resource Group Id %w", err)
			}
			options.ResourceGroupID = &rg
		}
		pgs, response, err := vpcclient.ListPublicGateways(options)
		if err != nil {
			return fmt.Errorf("error fetching Public Gateways %w\n%s", err, response)
		}
		start = GetNext(pgs.Next)
		allrecs = append(allrecs, pgs.PublicGateways...)
		if start == "" {
			break
		}
	}

	for _, pg := range allrecs {
		g.Resources = append(g.Resources, g.createPublicGatewayResources(*pg.ID, *pg.Name))
	}

	return nil
}
