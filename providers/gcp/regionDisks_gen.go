// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var regionDisksAllowEmptyValues = []string{""}

var regionDisksAdditionalFields = map[string]interface{}{}

type RegionDisksGenerator struct {
	GCPService
}

// Run on regionDisksList and create for each TerraformResource
func (g RegionDisksGenerator) createResources(ctx context.Context, regionDisksList *compute.RegionDisksListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := regionDisksList.Pages(ctx, func(page *compute.DiskList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_region_disk",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				regionDisksAllowEmptyValues,
				regionDisksAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list regionDisks: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each regionDisks create 1 TerraformResource
// Need regionDisks name as ID for terraform resource
func (g *RegionDisksGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	regionDisksList := computeService.RegionDisks.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, regionDisksList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
