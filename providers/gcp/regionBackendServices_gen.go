// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var regionBackendServicesAllowEmptyValues = []string{""}

var regionBackendServicesAdditionalFields = map[string]interface{}{}

type RegionBackendServicesGenerator struct {
	GCPService
}

// Run on regionBackendServicesList and create for each TerraformResource
func (g RegionBackendServicesGenerator) createResources(ctx context.Context, regionBackendServicesList *compute.RegionBackendServicesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := regionBackendServicesList.Pages(ctx, func(page *compute.BackendServiceList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_region_backend_service",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				regionBackendServicesAllowEmptyValues,
				regionBackendServicesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list regionBackendServices: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each regionBackendServices create 1 TerraformResource
// Need regionBackendServices name as ID for terraform resource
func (g *RegionBackendServicesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	regionBackendServicesList := computeService.RegionBackendServices.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, regionBackendServicesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
