// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var resourcePoliciesAllowEmptyValues = []string{""}

var resourcePoliciesAdditionalFields = map[string]interface{}{}

type ResourcePoliciesGenerator struct {
	GCPService
}

// Run on resourcePoliciesList and create for each TerraformResource
func (g ResourcePoliciesGenerator) createResources(ctx context.Context, resourcePoliciesList *compute.ResourcePoliciesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := resourcePoliciesList.Pages(ctx, func(page *compute.ResourcePolicyList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_resource_policy",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				resourcePoliciesAllowEmptyValues,
				resourcePoliciesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list resourcePolicies: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each resourcePolicies create 1 TerraformResource
// Need resourcePolicies name as ID for terraform resource
func (g *ResourcePoliciesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	resourcePoliciesList := computeService.ResourcePolicies.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, resourcePoliciesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
