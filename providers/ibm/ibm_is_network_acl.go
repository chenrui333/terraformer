// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"log"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// NetworkACLGenerator ...
type NetworkACLGenerator struct {
	IBMService
}

func (g NetworkACLGenerator) createNetworkACLResources(nwaclID, nwaclName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		nwaclID,
		normalizeResourceName(nwaclName, true),
		"ibm_is_network_acl",
		"ibm",
		[]string{})
	return resources
}

// InitResources ...
func (g *NetworkACLGenerator) InitResources() error {
	region := g.Args["region"].(string)
	apiKey := os.Getenv("IC_API_KEY")
	if apiKey == "" {
		log.Fatal("No API key set")
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
	var allrecs []vpcv1.NetworkACL
	for {
		options := &vpcv1.ListNetworkAclsOptions{}
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
		nwacls, response, err := vpcclient.ListNetworkAcls(options)
		if err != nil {
			return fmt.Errorf("error fetching Network ACLs %w\n%s", err, response)
		}
		start = GetNext(nwacls.Next)
		allrecs = append(allrecs, nwacls.NetworkAcls...)
		if start == "" {
			break
		}
	}

	for _, nwacl := range allrecs {
		g.Resources = append(g.Resources, g.createNetworkACLResources(*nwacl.ID, *nwacl.Name))
	}
	return nil
}
