// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var regionInstanceGroupManagersAllowEmptyValues = []string{"name", "health_check"}

var regionInstanceGroupManagersAdditionalFields = map[string]interface{}{}

type RegionInstanceGroupManagersGenerator struct {
	GCPService
}

// Run on regionInstanceGroupManagersList and create for each TerraformResource
func (g RegionInstanceGroupManagersGenerator) createResources(ctx context.Context, regionInstanceGroupManagersList *compute.RegionInstanceGroupManagersListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := regionInstanceGroupManagersList.Pages(ctx, func(page *compute.RegionInstanceGroupManagerList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_region_instance_group_manager",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				regionInstanceGroupManagersAllowEmptyValues,
				regionInstanceGroupManagersAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each regionInstanceGroupManagers create 1 TerraformResource
// Need regionInstanceGroupManagers name as ID for terraform resource
func (g *RegionInstanceGroupManagersGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	regionInstanceGroupManagersList := computeService.RegionInstanceGroupManagers.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	g.Resources = g.createResources(ctx, regionInstanceGroupManagersList)

	return nil

}
