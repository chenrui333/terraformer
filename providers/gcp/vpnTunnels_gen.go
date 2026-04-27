// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var vpnTunnelsAllowEmptyValues = []string{""}

var vpnTunnelsAdditionalFields = map[string]interface{}{}

type VpnTunnelsGenerator struct {
	GCPService
}

// Run on vpnTunnelsList and create for each TerraformResource
func (g VpnTunnelsGenerator) createResources(ctx context.Context, vpnTunnelsList *compute.VpnTunnelsListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := vpnTunnelsList.Pages(ctx, func(page *compute.VpnTunnelList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_vpn_tunnel",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				vpnTunnelsAllowEmptyValues,
				vpnTunnelsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each vpnTunnels create 1 TerraformResource
// Need vpnTunnels name as ID for terraform resource
func (g *VpnTunnelsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	vpnTunnelsList := computeService.VpnTunnels.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	g.Resources = g.createResources(ctx, vpnTunnelsList)

	return nil

}
