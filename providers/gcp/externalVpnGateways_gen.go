// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var externalVpnGatewaysAllowEmptyValues = []string{""}

var externalVpnGatewaysAdditionalFields = map[string]interface{}{}

type ExternalVpnGatewaysGenerator struct {
	GCPService
}

// Run on externalVpnGatewaysList and create for each TerraformResource
func (g ExternalVpnGatewaysGenerator) createResources(ctx context.Context, externalVpnGatewaysList *compute.ExternalVpnGatewaysListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := externalVpnGatewaysList.Pages(ctx, func(page *compute.ExternalVpnGatewayList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_external_vpn_gateway",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				externalVpnGatewaysAllowEmptyValues,
				externalVpnGatewaysAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list externalVpnGateways: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each externalVpnGateways create 1 TerraformResource
// Need externalVpnGateways name as ID for terraform resource
func (g *ExternalVpnGatewaysGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	externalVpnGatewaysList := computeService.ExternalVpnGateways.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, externalVpnGatewaysList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
