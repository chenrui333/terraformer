// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var securityPoliciesAllowEmptyValues = []string{""}

var securityPoliciesAdditionalFields = map[string]interface{}{}

type SecurityPoliciesGenerator struct {
	GCPService
}

// Run on securityPoliciesList and create for each TerraformResource
func (g SecurityPoliciesGenerator) createResources(ctx context.Context, securityPoliciesList *compute.SecurityPoliciesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := securityPoliciesList.Pages(ctx, func(page *compute.SecurityPolicyList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_security_policy",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				securityPoliciesAllowEmptyValues,
				securityPoliciesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list securityPolicies: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each securityPolicies create 1 TerraformResource
// Need securityPolicies name as ID for terraform resource
func (g *SecurityPoliciesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	securityPoliciesList := computeService.SecurityPolicies.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, securityPoliciesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
