// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var healthChecksAllowEmptyValues = []string{""}

var healthChecksAdditionalFields = map[string]interface{}{}

type HealthChecksGenerator struct {
	GCPService
}

// Run on healthChecksList and create for each TerraformResource
func (g HealthChecksGenerator) createResources(ctx context.Context, healthChecksList *compute.HealthChecksListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := healthChecksList.Pages(ctx, func(page *compute.HealthCheckList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_health_check",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				healthChecksAllowEmptyValues,
				healthChecksAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each healthChecks create 1 TerraformResource
// Need healthChecks name as ID for terraform resource
func (g *HealthChecksGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	healthChecksList := computeService.HealthChecks.List(g.GetArgs()["project"].(string))
	g.Resources = g.createResources(ctx, healthChecksList)

	return nil

}
