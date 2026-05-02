// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var httpHealthChecksAllowEmptyValues = []string{""}

var httpHealthChecksAdditionalFields = map[string]interface{}{}

type HttpHealthChecksGenerator struct {
	GCPService
}

// Run on httpHealthChecksList and create for each TerraformResource
func (g HttpHealthChecksGenerator) createResources(ctx context.Context, httpHealthChecksList *compute.HttpHealthChecksListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := httpHealthChecksList.Pages(ctx, func(page *compute.HttpHealthCheckList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_http_health_check",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				httpHealthChecksAllowEmptyValues,
				httpHealthChecksAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list httpHealthChecks: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each httpHealthChecks create 1 TerraformResource
// Need httpHealthChecks name as ID for terraform resource
func (g *HttpHealthChecksGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	httpHealthChecksList := computeService.HttpHealthChecks.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, httpHealthChecksList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
