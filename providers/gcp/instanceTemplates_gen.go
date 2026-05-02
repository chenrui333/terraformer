// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var instanceTemplatesAllowEmptyValues = []string{""}

var instanceTemplatesAdditionalFields = map[string]interface{}{}

type InstanceTemplatesGenerator struct {
	GCPService
}

// Run on instanceTemplatesList and create for each TerraformResource
func (g InstanceTemplatesGenerator) createResources(ctx context.Context, instanceTemplatesList *compute.InstanceTemplatesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := instanceTemplatesList.Pages(ctx, func(page *compute.InstanceTemplateList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_instance_template",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				instanceTemplatesAllowEmptyValues,
				instanceTemplatesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list instanceTemplates: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each instanceTemplates create 1 TerraformResource
// Need instanceTemplates name as ID for terraform resource
func (g *InstanceTemplatesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	instanceTemplatesList := computeService.InstanceTemplates.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, instanceTemplatesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
