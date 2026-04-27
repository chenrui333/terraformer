// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// VPCGenerator ...
type VPCGenerator struct {
	IBMService
}

func (g VPCGenerator) createVPCResources(vpcID, vpcName string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		vpcID,
		normalizeResourceName(vpcName, false),
		"ibm_is_vpc",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{})

	// Deprecated parameters
	resource.IgnoreKeys = append(resource.IgnoreKeys,
		"^default_network_acl$",
	)

	return resource
}

// InitResources ...
func (g *VPCGenerator) InitResources() error {
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
		g.Resources = append(g.Resources, g.createVPCResources(*vpc.ID, *vpc.Name))
	}
	return nil
}
