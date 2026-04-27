// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var targetVpnGatewaysAllowEmptyValues = []string{""}

var targetVpnGatewaysAdditionalFields = map[string]interface{}{}

type TargetVpnGatewaysGenerator struct {
	GCPService
}

// Run on targetVpnGatewaysList and create for each TerraformResource
func (g TargetVpnGatewaysGenerator) createResources(ctx context.Context, targetVpnGatewaysList *compute.TargetVpnGatewaysListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := targetVpnGatewaysList.Pages(ctx, func(page *compute.TargetVpnGatewayList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_vpn_gateway",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				targetVpnGatewaysAllowEmptyValues,
				targetVpnGatewaysAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each targetVpnGateways create 1 TerraformResource
// Need targetVpnGateways name as ID for terraform resource
func (g *TargetVpnGatewaysGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	targetVpnGatewaysList := computeService.TargetVpnGateways.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	g.Resources = g.createResources(ctx, targetVpnGatewaysList)

	return nil

}
