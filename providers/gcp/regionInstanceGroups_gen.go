// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var regionInstanceGroupsAllowEmptyValues = []string{""}

var regionInstanceGroupsAdditionalFields = map[string]interface{}{}

type RegionInstanceGroupsGenerator struct {
	GCPService
}

// Run on regionInstanceGroupsList and create for each TerraformResource
func (g RegionInstanceGroupsGenerator) createResources(ctx context.Context, regionInstanceGroupsList *compute.RegionInstanceGroupsListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := regionInstanceGroupsList.Pages(ctx, func(page *compute.RegionInstanceGroupList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_region_instance_group",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				regionInstanceGroupsAllowEmptyValues,
				regionInstanceGroupsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list regionInstanceGroups: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each regionInstanceGroups create 1 TerraformResource
// Need regionInstanceGroups name as ID for terraform resource
func (g *RegionInstanceGroupsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	regionInstanceGroupsList := computeService.RegionInstanceGroups.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, regionInstanceGroupsList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
