// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var imagesAllowEmptyValues = []string{""}

var imagesAdditionalFields = map[string]interface{}{}

type ImagesGenerator struct {
	GCPService
}

// Run on imagesList and create for each TerraformResource
func (g ImagesGenerator) createResources(ctx context.Context, imagesList *compute.ImagesListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := imagesList.Pages(ctx, func(page *compute.ImageList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_image",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				imagesAllowEmptyValues,
				imagesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each images create 1 TerraformResource
// Need images name as ID for terraform resource
func (g *ImagesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	imagesList := computeService.Images.List(g.GetArgs()["project"].(string))
	g.Resources = g.createResources(ctx, imagesList)

	return nil

}
