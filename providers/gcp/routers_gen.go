// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var routersAllowEmptyValues = []string{""}

var routersAdditionalFields = map[string]interface{}{}

type RoutersGenerator struct {
	GCPService
}

// Run on routersList and create for each TerraformResource
func (g RoutersGenerator) createResources(ctx context.Context, routersList *compute.RoutersListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := routersList.Pages(ctx, func(page *compute.RouterList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_router",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				routersAllowEmptyValues,
				routersAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list routers: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each routers create 1 TerraformResource
// Need routers name as ID for terraform resource
func (g *RoutersGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	routersList := computeService.Routers.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, routersList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
