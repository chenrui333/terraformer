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

// IkeGenerator ...
type IkeGenerator struct {
	IBMService
}

func (g IkeGenerator) createIkeResources(ikeID, ikeName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		ikeID,
		normalizeResourceName(ikeName, false),
		"ibm_is_ike_policy",
		"ibm",
		[]string{})
	return resources
}

// InitResources ...
func (g *IkeGenerator) InitResources() error {
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
	var allrecs []vpcv1.IkePolicy
	for {
		options := &vpcv1.ListIkePoliciesOptions{}
		if start != "" {
			options.Start = &start
		}
		policies, response, err := vpcclient.ListIkePolicies(options)
		if err != nil {
			return fmt.Errorf("error fetching IKE Policies %w\n%s", err, response)
		}
		start = GetNext(policies.Next)
		allrecs = append(allrecs, policies.IkePolicies...)
		if start == "" {
			break
		}
	}

	for _, policy := range allrecs {
		g.Resources = append(g.Resources, g.createIkeResources(*policy.ID, *policy.Name))
	}
	return nil
}
