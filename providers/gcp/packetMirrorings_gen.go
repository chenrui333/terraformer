// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var packetMirroringsAllowEmptyValues = []string{""}

var packetMirroringsAdditionalFields = map[string]interface{}{}

type PacketMirroringsGenerator struct {
	GCPService
}

// Run on packetMirroringsList and create for each TerraformResource
func (g PacketMirroringsGenerator) createResources(ctx context.Context, packetMirroringsList *compute.PacketMirroringsListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := packetMirroringsList.Pages(ctx, func(page *compute.PacketMirroringList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_packet_mirroring",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				packetMirroringsAllowEmptyValues,
				packetMirroringsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list packetMirrorings: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each packetMirrorings create 1 TerraformResource
// Need packetMirrorings name as ID for terraform resource
func (g *PacketMirroringsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	packetMirroringsList := computeService.PacketMirrorings.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, packetMirroringsList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
