// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var regionHealthChecksAllowEmptyValues = []string{""}

var regionHealthChecksAdditionalFields = map[string]interface{}{}

type RegionHealthChecksGenerator struct {
	GCPService
}

// Run on regionHealthChecksList and create for each TerraformResource
func (g RegionHealthChecksGenerator) createResources(ctx context.Context, regionHealthChecksList *compute.RegionHealthChecksListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := regionHealthChecksList.Pages(ctx, func(page *compute.HealthCheckList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_region_health_check",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				regionHealthChecksAllowEmptyValues,
				regionHealthChecksAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list regionHealthChecks: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each regionHealthChecks create 1 TerraformResource
// Need regionHealthChecks name as ID for terraform resource
func (g *RegionHealthChecksGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	regionHealthChecksList := computeService.RegionHealthChecks.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, regionHealthChecksList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
