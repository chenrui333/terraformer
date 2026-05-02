// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var sslPoliciesAllowEmptyValues = []string{""}

var sslPoliciesAdditionalFields = map[string]interface{}{}

type SslPoliciesGenerator struct {
	GCPService
}

// Run on sslPoliciesList and create for each TerraformResource
func (g SslPoliciesGenerator) createResources(ctx context.Context, sslPoliciesList *compute.SslPoliciesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := sslPoliciesList.Pages(ctx, func(page *compute.SslPoliciesList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_ssl_policy",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				sslPoliciesAllowEmptyValues,
				sslPoliciesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list sslPolicies: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each sslPolicies create 1 TerraformResource
// Need sslPolicies name as ID for terraform resource
func (g *SslPoliciesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	sslPoliciesList := computeService.SslPolicies.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, sslPoliciesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
