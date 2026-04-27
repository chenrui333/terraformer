// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var globalAddressesAllowEmptyValues = []string{""}

var globalAddressesAdditionalFields = map[string]interface{}{}

type GlobalAddressesGenerator struct {
	GCPService
}

// Run on globalAddressesList and create for each TerraformResource
func (g GlobalAddressesGenerator) createResources(ctx context.Context, globalAddressesList *compute.GlobalAddressesListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := globalAddressesList.Pages(ctx, func(page *compute.AddressList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_global_address",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				globalAddressesAllowEmptyValues,
				globalAddressesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each globalAddresses create 1 TerraformResource
// Need globalAddresses name as ID for terraform resource
func (g *GlobalAddressesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	globalAddressesList := computeService.GlobalAddresses.List(g.GetArgs()["project"].(string))
	g.Resources = g.createResources(ctx, globalAddressesList)

	return nil

}
