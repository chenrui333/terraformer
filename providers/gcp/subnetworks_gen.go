// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var subnetworksAllowEmptyValues = []string{""}

var subnetworksAdditionalFields = map[string]interface{}{}

type SubnetworksGenerator struct {
	GCPService
}

// Run on subnetworksList and create for each TerraformResource
func (g SubnetworksGenerator) createResources(ctx context.Context, subnetworksList *compute.SubnetworksListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := subnetworksList.Pages(ctx, func(page *compute.SubnetworkList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_subnetwork",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				subnetworksAllowEmptyValues,
				subnetworksAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list subnetworks: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each subnetworks create 1 TerraformResource
// Need subnetworks name as ID for terraform resource
func (g *SubnetworksGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	subnetworksList := computeService.Subnetworks.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, subnetworksList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
