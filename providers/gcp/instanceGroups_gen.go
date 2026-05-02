// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var instanceGroupsAllowEmptyValues = []string{""}

var instanceGroupsAdditionalFields = map[string]interface{}{}

type InstanceGroupsGenerator struct {
	GCPService
}

// Run on instanceGroupsList and create for each TerraformResource
func (g InstanceGroupsGenerator) createResources(ctx context.Context, instanceGroupsList *compute.InstanceGroupsListCall, zone string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := instanceGroupsList.Pages(ctx, func(page *compute.InstanceGroupList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				zone+"/"+obj.Name,
				zone+"/"+obj.Name,
				"google_compute_instance_group",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
					"zone":    zone,
				},
				instanceGroupsAllowEmptyValues,
				instanceGroupsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list instanceGroups: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each instanceGroups create 1 TerraformResource
// Need instanceGroups name as ID for terraform resource
func (g *InstanceGroupsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	for _, zoneLink := range g.GetArgs()["region"].(compute.Region).Zones {
		t := strings.Split(zoneLink, "/")
		zone := t[len(t)-1]
		instanceGroupsList := computeService.InstanceGroups.List(g.GetArgs()["project"].(string), zone)
		resources, err := g.createResources(ctx, instanceGroupsList, zone)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	return nil

}
