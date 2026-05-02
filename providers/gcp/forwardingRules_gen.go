// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var forwardingRulesAllowEmptyValues = []string{""}

var forwardingRulesAdditionalFields = map[string]interface{}{}

type ForwardingRulesGenerator struct {
	GCPService
}

// Run on forwardingRulesList and create for each TerraformResource
func (g ForwardingRulesGenerator) createResources(ctx context.Context, forwardingRulesList *compute.ForwardingRulesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := forwardingRulesList.Pages(ctx, func(page *compute.ForwardingRuleList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_forwarding_rule",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				forwardingRulesAllowEmptyValues,
				forwardingRulesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list forwardingRules: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each forwardingRules create 1 TerraformResource
// Need forwardingRules name as ID for terraform resource
func (g *ForwardingRulesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	forwardingRulesList := computeService.ForwardingRules.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, forwardingRulesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
