// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var networkEndpointGroupsAllowEmptyValues = []string{""}

var networkEndpointGroupsAdditionalFields = map[string]interface{}{}

type NetworkEndpointGroupsGenerator struct {
	GCPService
}

// Run on networkEndpointGroupsList and create for each TerraformResource
func (g NetworkEndpointGroupsGenerator) createResources(ctx context.Context, networkEndpointGroupsList *compute.NetworkEndpointGroupsListCall, zone string) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := networkEndpointGroupsList.Pages(ctx, func(page *compute.NetworkEndpointGroupList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				zone+"/"+obj.Name,
				zone+"/"+obj.Name,
				"google_compute_network_endpoint_group",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
					"zone":    zone,
				},
				networkEndpointGroupsAllowEmptyValues,
				networkEndpointGroupsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each networkEndpointGroups create 1 TerraformResource
// Need networkEndpointGroups name as ID for terraform resource
func (g *NetworkEndpointGroupsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	for _, zoneLink := range g.GetArgs()["region"].(compute.Region).Zones {
		t := strings.Split(zoneLink, "/")
		zone := t[len(t)-1]
		networkEndpointGroupsList := computeService.NetworkEndpointGroups.List(g.GetArgs()["project"].(string), zone)
		g.Resources = append(g.Resources, g.createResources(ctx, networkEndpointGroupsList, zone)...)
	}

	return nil

}
