// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// FlowLogGenerator ...
type FlowLogGenerator struct {
	IBMService
}

func (g FlowLogGenerator) createFlowLogResources(flogID, flogName string) terraformutils.Resource {
	resource := terraformutils.NewSimpleResource(
		flogID,
		normalizeResourceName(flogName, false),
		"ibm_is_flow_log",
		"ibm",
		[]string{})
	return resource
}

// InitResources ...
func (g *FlowLogGenerator) InitResources() error {
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
	var allrecs []vpcv1.FlowLogCollector
	for {
		options := &vpcv1.ListFlowLogCollectorsOptions{}
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
		flogs, response, err := vpcclient.ListFlowLogCollectors(options)
		if err != nil {
			return fmt.Errorf("error fetching Flow Logs %w\n%s", err, response)
		}
		start = GetNext(flogs.Next)
		allrecs = append(allrecs, flogs.FlowLogCollectors...)
		if start == "" {
			break
		}
	}

	for _, flog := range allrecs {
		g.Resources = append(g.Resources, g.createFlowLogResources(*flog.ID, *flog.Name))
	}
	return nil
}
