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

var instanceGroupManagersAllowEmptyValues = []string{"^version.[0-9].name", "^auto_healing_policies.[0-9].health_check"}

var instanceGroupManagersAdditionalFields = map[string]interface{}{}

type InstanceGroupManagersGenerator struct {
	GCPService
}

// Run on instanceGroupManagersList and create for each TerraformResource
func (g InstanceGroupManagersGenerator) createResources(ctx context.Context, instanceGroupManagersList *compute.InstanceGroupManagersListCall, zone string) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := instanceGroupManagersList.Pages(ctx, func(page *compute.InstanceGroupManagerList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_instance_group_manager",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),

					"zone": zone,
				},
				instanceGroupManagersAllowEmptyValues,
				instanceGroupManagersAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each instanceGroupManagers create 1 TerraformResource
// Need instanceGroupManagers name as ID for terraform resource
func (g *InstanceGroupManagersGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	for _, zoneLink := range g.GetArgs()["region"].(compute.Region).Zones {
		t := strings.Split(zoneLink, "/")
		zone := t[len(t)-1]
		instanceGroupManagersList := computeService.InstanceGroupManagers.List(g.GetArgs()["project"].(string), zone)
		g.Resources = append(g.Resources, g.createResources(ctx, instanceGroupManagersList, zone)...)
	}

	return nil

}
