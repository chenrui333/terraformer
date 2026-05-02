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

var targetInstancesAllowEmptyValues = []string{""}

var targetInstancesAdditionalFields = map[string]interface{}{}

type TargetInstancesGenerator struct {
	GCPService
}

// Run on targetInstancesList and create for each TerraformResource
func (g TargetInstancesGenerator) createResources(ctx context.Context, targetInstancesList *compute.TargetInstancesListCall, zone string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := targetInstancesList.Pages(ctx, func(page *compute.TargetInstanceList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				zone+"/"+obj.Name,
				zone+"/"+obj.Name,
				"google_compute_target_instance",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
					"zone":    zone,
				},
				targetInstancesAllowEmptyValues,
				targetInstancesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list targetInstances: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each targetInstances create 1 TerraformResource
// Need targetInstances name as ID for terraform resource
func (g *TargetInstancesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	for _, zoneLink := range g.GetArgs()["region"].(compute.Region).Zones {
		t := strings.Split(zoneLink, "/")
		zone := t[len(t)-1]
		targetInstancesList := computeService.TargetInstances.List(g.GetArgs()["project"].(string), zone)
		resources, err := g.createResources(ctx, targetInstancesList, zone)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	return nil

}
