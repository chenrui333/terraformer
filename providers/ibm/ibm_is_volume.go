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

// VolumeGenerator ...
type VolumeGenerator struct {
	IBMService
}

func (g VolumeGenerator) createVolumeResources(volID, volName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		volID,
		normalizeResourceName(volName, true),
		"ibm_is_volume",
		"ibm",
		[]string{})
	return resources
}

// InitResources ...
func (g *VolumeGenerator) InitResources() error {
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
	var allrecs []vpcv1.Volume
	for {
		options := &vpcv1.ListVolumesOptions{}
		if start != "" {
			options.Start = &start
		}
		volumes, response, err := vpcclient.ListVolumes(options)
		if err != nil {
			return fmt.Errorf("error fetching Volumes %w\n%s", err, response)
		}
		start = GetNext(volumes.Next)
		allrecs = append(allrecs, volumes.Volumes...)
		if start == "" {
			break
		}
	}

	for _, volume := range allrecs {
		g.Resources = append(g.Resources, g.createVolumeResources(*volume.ID, *volume.Name))
	}
	return nil
}
