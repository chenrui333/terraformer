// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var routesAllowEmptyValues = []string{""}

var routesAdditionalFields = map[string]interface{}{}

type RoutesGenerator struct {
	GCPService
}

// Run on routesList and create for each TerraformResource
func (g RoutesGenerator) createResources(ctx context.Context, routesList *compute.RoutesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := routesList.Pages(ctx, func(page *compute.RouteList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_route",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				routesAllowEmptyValues,
				routesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each routes create 1 TerraformResource
// Need routes name as ID for terraform resource
func (g *RoutesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	routesList := computeService.Routes.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, routesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
