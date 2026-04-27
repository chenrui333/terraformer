// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// VPCRouteGenerator ...
type VPCRouteGenerator struct {
	IBMService
}

func (g VPCRouteGenerator) loadVPCRouteResources(vpcID, routeID, routeName string) terraformutils.Resource {
	resources := terraformutils.NewResource(
		fmt.Sprintf("%s/%s", vpcID, routeID),
		normalizeResourceName(routeName, false),
		"ibm_is_vpc_route",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{})

	return resources
}

// InitResources ...
func (g *VPCRouteGenerator) InitResources() error {
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
	var allrecs []vpcv1.VPC
	for {
		listVpcsOptions := &vpcv1.ListVpcsOptions{}
		if start != "" {
			listVpcsOptions.Start = &start
		}
		if rg := g.Args["resource_group"].(string); rg != "" {
			rg, err = GetResourceGroupID(apiKey, rg, region)
			if err != nil {
				return fmt.Errorf("error fetching Resource Group Id %w", err)
			}
			listVpcsOptions.ResourceGroupID = &rg
		}
		vpcs, response, err := vpcclient.ListVpcs(listVpcsOptions)
		if err != nil {
			return fmt.Errorf("error fetching vpcs %w\n%s", err, response)
		}
		start = GetNext(vpcs.Next)
		allrecs = append(allrecs, vpcs.Vpcs...)
		if start == "" {
			break
		}
	}

	for _, vpc := range allrecs {
		listVPCRoutesOptions := &vpcv1.ListVPCRoutesOptions{
			VPCID: vpc.ID,
		}
		routes, response, err := vpcclient.ListVPCRoutes(listVPCRoutesOptions)
		if err != nil {
			return fmt.Errorf("error fetching vpc routes %w\n%s", err, response)
		}
		for _, route := range routes.Routes {
			g.Resources = append(g.Resources, g.loadVPCRouteResources(*vpc.ID, *route.ID, *route.Name))
		}
	}

	return nil
}
