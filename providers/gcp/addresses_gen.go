// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var addressesAllowEmptyValues = []string{""}

var addressesAdditionalFields = map[string]interface{}{}

type AddressesGenerator struct {
	GCPService
}

// Run on addressesList and create for each TerraformResource
func (g AddressesGenerator) createResources(ctx context.Context, addressesList *compute.AddressesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := addressesList.Pages(ctx, func(page *compute.AddressList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_address",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				addressesAllowEmptyValues,
				addressesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each addresses create 1 TerraformResource
// Need addresses name as ID for terraform resource
func (g *AddressesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	addressesList := computeService.Addresses.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, addressesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
