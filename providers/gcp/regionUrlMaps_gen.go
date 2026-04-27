// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var regionUrlMapsAllowEmptyValues = []string{""}

var regionUrlMapsAdditionalFields = map[string]interface{}{}

type RegionUrlMapsGenerator struct {
	GCPService
}

// Run on regionUrlMapsList and create for each TerraformResource
func (g RegionUrlMapsGenerator) createResources(ctx context.Context, regionUrlMapsList *compute.RegionUrlMapsListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := regionUrlMapsList.Pages(ctx, func(page *compute.UrlMapList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_region_url_map",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				regionUrlMapsAllowEmptyValues,
				regionUrlMapsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each regionUrlMaps create 1 TerraformResource
// Need regionUrlMaps name as ID for terraform resource
func (g *RegionUrlMapsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	regionUrlMapsList := computeService.RegionUrlMaps.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	g.Resources = g.createResources(ctx, regionUrlMapsList)

	return nil

}
