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

// ImageGenerator ...
type ImageGenerator struct {
	IBMService
}

func (g ImageGenerator) createImageResources(imageID, imageName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		imageID,
		normalizeResourceName(imageName, true),
		"ibm_is_image",
		"ibm",
		[]string{})
	return resources
}

// InitResources ...
func (g *ImageGenerator) InitResources() error {
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
	var allrecs []vpcv1.Image
	for {
		options := &vpcv1.ListImagesOptions{}
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
		images, response, err := vpcclient.ListImages(options)
		if err != nil {
			return fmt.Errorf("error fetching Images %w\n%s", err, response)
		}
		start = GetNext(images.Next)
		allrecs = append(allrecs, images.Images...)
		if start == "" {
			break
		}
	}

	for _, image := range allrecs {
		g.Resources = append(g.Resources, g.createImageResources(*image.ID, *image.Name))
	}
	return nil
}
