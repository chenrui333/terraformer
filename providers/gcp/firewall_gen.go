// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var firewallAllowEmptyValues = []string{""}

var firewallAdditionalFields = map[string]interface{}{}

type FirewallGenerator struct {
	GCPService
}

// Run on firewallList and create for each TerraformResource
func (g FirewallGenerator) createResources(ctx context.Context, firewallList *compute.FirewallsListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := firewallList.Pages(ctx, func(page *compute.FirewallList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_firewall",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				firewallAllowEmptyValues,
				firewallAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list firewall: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each firewall create 1 TerraformResource
// Need firewall name as ID for terraform resource
func (g *FirewallGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	firewallList := computeService.Firewalls.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, firewallList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
