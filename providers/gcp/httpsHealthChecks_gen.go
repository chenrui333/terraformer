// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var httpsHealthChecksAllowEmptyValues = []string{""}

var httpsHealthChecksAdditionalFields = map[string]interface{}{}

type HttpsHealthChecksGenerator struct {
	GCPService
}

// Run on httpsHealthChecksList and create for each TerraformResource
func (g HttpsHealthChecksGenerator) createResources(ctx context.Context, httpsHealthChecksList *compute.HttpsHealthChecksListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := httpsHealthChecksList.Pages(ctx, func(page *compute.HttpsHealthCheckList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_https_health_check",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				httpsHealthChecksAllowEmptyValues,
				httpsHealthChecksAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list httpsHealthChecks: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each httpsHealthChecks create 1 TerraformResource
// Need httpsHealthChecks name as ID for terraform resource
func (g *HttpsHealthChecksGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	httpsHealthChecksList := computeService.HttpsHealthChecks.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, httpsHealthChecksList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
