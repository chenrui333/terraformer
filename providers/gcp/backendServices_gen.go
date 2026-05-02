// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var backendServicesAllowEmptyValues = []string{""}

var backendServicesAdditionalFields = map[string]interface{}{}

type BackendServicesGenerator struct {
	GCPService
}

// Run on backendServicesList and create for each TerraformResource
func (g BackendServicesGenerator) createResources(ctx context.Context, backendServicesList *compute.BackendServicesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := backendServicesList.Pages(ctx, func(page *compute.BackendServiceList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_backend_service",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
				},
				backendServicesAllowEmptyValues,
				backendServicesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list backendServices: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each backendServices create 1 TerraformResource
// Need backendServices name as ID for terraform resource
func (g *BackendServicesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	backendServicesList := computeService.BackendServices.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, backendServicesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
