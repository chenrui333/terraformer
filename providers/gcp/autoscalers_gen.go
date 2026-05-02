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

var autoscalersAllowEmptyValues = []string{""}

var autoscalersAdditionalFields = map[string]interface{}{}

type AutoscalersGenerator struct {
	GCPService
}

// Run on autoscalersList and create for each TerraformResource
func (g AutoscalersGenerator) createResources(ctx context.Context, autoscalersList *compute.AutoscalersListCall, zone string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := autoscalersList.Pages(ctx, func(page *compute.AutoscalerList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				zone+"/"+obj.Name,
				zone+"/"+obj.Name,
				"google_compute_autoscaler",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
					"zone":    zone,
				},
				autoscalersAllowEmptyValues,
				autoscalersAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list autoscalers: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each autoscalers create 1 TerraformResource
// Need autoscalers name as ID for terraform resource
func (g *AutoscalersGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	for _, zoneLink := range g.GetArgs()["region"].(compute.Region).Zones {
		t := strings.Split(zoneLink, "/")
		zone := t[len(t)-1]
		autoscalersList := computeService.Autoscalers.List(g.GetArgs()["project"].(string), zone)
		resources, err := g.createResources(ctx, autoscalersList, zone)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	return nil

}
