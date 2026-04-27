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
type VPCAddressPrefixGenerator struct {
	IBMService
}

func (g VPCAddressPrefixGenerator) createVPCAddressPrefixResources(vpcID, addPrefixID, addPrefixName string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		fmt.Sprintf("%s/%s", vpcID, addPrefixID),
		normalizeResourceName(addPrefixName, false),
		"ibm_is_vpc_address_prefix",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{})

	return resource
}

// InitResources ...
func (g *VPCAddressPrefixGenerator) InitResources() error {
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
		// address prefix
		listVPCAddressPrefixesOptions := &vpcv1.ListVPCAddressPrefixesOptions{
			VPCID: vpc.ID,
		}
		addprefixes, response, err := vpcclient.ListVPCAddressPrefixes(listVPCAddressPrefixesOptions)
		if err != nil {
			return fmt.Errorf("error fetching vpc address prefixes %w\n%s", err, response)
		}
		for _, addprefix := range addprefixes.AddressPrefixes {
			g.Resources = append(g.Resources, g.createVPCAddressPrefixResources(*vpc.ID, *addprefix.ID, *addprefix.Name))
		}
	}
	return nil
}
