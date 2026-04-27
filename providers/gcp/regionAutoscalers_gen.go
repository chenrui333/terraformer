// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var regionAutoscalersAllowEmptyValues = []string{""}

var regionAutoscalersAdditionalFields = map[string]interface{}{}

type RegionAutoscalersGenerator struct {
	GCPService
}

// Run on regionAutoscalersList and create for each TerraformResource
func (g RegionAutoscalersGenerator) createResources(ctx context.Context, regionAutoscalersList *compute.RegionAutoscalersListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := regionAutoscalersList.Pages(ctx, func(page *compute.RegionAutoscalerList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_region_autoscaler",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				regionAutoscalersAllowEmptyValues,
				regionAutoscalersAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each regionAutoscalers create 1 TerraformResource
// Need regionAutoscalers name as ID for terraform resource
func (g *RegionAutoscalersGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	regionAutoscalersList := computeService.RegionAutoscalers.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	g.Resources = g.createResources(ctx, regionAutoscalersList)

	return nil

}
