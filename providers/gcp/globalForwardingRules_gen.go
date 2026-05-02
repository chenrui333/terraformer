// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var globalForwardingRulesAllowEmptyValues = []string{""}

var globalForwardingRulesAdditionalFields = map[string]interface{}{}

type GlobalForwardingRulesGenerator struct {
	GCPService
}

// Run on globalForwardingRulesList and create for each TerraformResource
func (g GlobalForwardingRulesGenerator) createResources(ctx context.Context, globalForwardingRulesList *compute.GlobalForwardingRulesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := globalForwardingRulesList.Pages(ctx, func(page *compute.ForwardingRuleList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_global_forwarding_rule",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
				},
				globalForwardingRulesAllowEmptyValues,
				globalForwardingRulesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list globalForwardingRules: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each globalForwardingRules create 1 TerraformResource
// Need globalForwardingRules name as ID for terraform resource
func (g *GlobalForwardingRulesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	globalForwardingRulesList := computeService.GlobalForwardingRules.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, globalForwardingRulesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
