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

// SubnetGenerator ...
type SubnetGenerator struct {
	IBMService
}

func (g SubnetGenerator) createSubnetResources(subnetID, subnetName string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		subnetID,
		normalizeResourceName(subnetName, true),
		"ibm_is_subnet",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{})

	resource.IgnoreKeys = append(resource.IgnoreKeys,
		"^total_ipv4_address_count$",
	)
	return resource
}

// InitResources ...
func (g *SubnetGenerator) InitResources() error {
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
		start = ""
		var allSubNetRecs []vpcv1.Subnet
		for {
			options := &vpcv1.ListSubnetsOptions{}
			if start != "" {
				options.Start = &start
			}

			subnets, response, err := vpcclient.ListSubnets(options)
			if err != nil {
				return fmt.Errorf("error fetching subnets %w\n%s", err, response)
			}
			start = GetNext(subnets.Next)
			allSubNetRecs = append(allSubNetRecs, subnets.Subnets...)
			if start == "" {
				break
			}
		}

		for _, subnet := range allSubNetRecs {
			if *vpc.ID == *subnet.VPC.ID {
				g.Resources = append(g.Resources, g.createSubnetResources(*subnet.ID, *subnet.Name))
			}
		}
	}

	return nil
}
