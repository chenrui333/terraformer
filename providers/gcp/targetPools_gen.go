// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var targetPoolsAllowEmptyValues = []string{""}

var targetPoolsAdditionalFields = map[string]interface{}{}

type TargetPoolsGenerator struct {
	GCPService
}

// Run on targetPoolsList and create for each TerraformResource
func (g TargetPoolsGenerator) createResources(ctx context.Context, targetPoolsList *compute.TargetPoolsListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := targetPoolsList.Pages(ctx, func(page *compute.TargetPoolList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_target_pool",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				targetPoolsAllowEmptyValues,
				targetPoolsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each targetPools create 1 TerraformResource
// Need targetPools name as ID for terraform resource
func (g *TargetPoolsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	targetPoolsList := computeService.TargetPools.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	g.Resources = g.createResources(ctx, targetPoolsList)

	return nil

}
