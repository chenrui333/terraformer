// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// IpsecGenerator ...
type IpsecGenerator struct {
	IBMService
}

func (g IpsecGenerator) createIpsecResources() func(ipsecID, ipsecName string) terraformutils.Resource {
	names := make(map[string]struct{})
	random := false
	return func(ipsecID, ipsecName string) terraformutils.Resource {
		names, random = getRandom(names, ipsecName, random)
		resources := terraformutils.NewSimpleResource(
			ipsecID,
			normalizeResourceName(ipsecName, random),
			"ibm_is_ipsec_policy",
			"ibm",
			[]string{})
		return resources
	}
}

// InitResources ...
func (g *IpsecGenerator) InitResources() error {
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
	var allrecs []vpcv1.IPsecPolicy
	for {
		options := &vpcv1.ListIpsecPoliciesOptions{}
		if start != "" {
			options.Start = &start
		}
		policies, response, err := vpcclient.ListIpsecPolicies(options)
		if err != nil {
			return fmt.Errorf("error fetching IPSEC Policies %w\n%s", err, response)
		}
		start = GetNext(policies.Next)
		allrecs = append(allrecs, policies.IpsecPolicies...)
		if start == "" {
			break
		}
	}

	fnObjt := g.createIpsecResources()
	for _, policy := range allrecs {
		g.Resources = append(g.Resources, fnObjt(*policy.ID, *policy.Name))
	}
	return nil
}
