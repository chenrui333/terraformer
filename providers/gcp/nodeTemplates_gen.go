// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var nodeTemplatesAllowEmptyValues = []string{""}

var nodeTemplatesAdditionalFields = map[string]interface{}{}

type NodeTemplatesGenerator struct {
	GCPService
}

// Run on nodeTemplatesList and create for each TerraformResource
func (g NodeTemplatesGenerator) createResources(ctx context.Context, nodeTemplatesList *compute.NodeTemplatesListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := nodeTemplatesList.Pages(ctx, func(page *compute.NodeTemplateList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_node_template",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				nodeTemplatesAllowEmptyValues,
				nodeTemplatesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each nodeTemplates create 1 TerraformResource
// Need nodeTemplates name as ID for terraform resource
func (g *NodeTemplatesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	nodeTemplatesList := computeService.NodeTemplates.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	g.Resources = g.createResources(ctx, nodeTemplatesList)

	return nil

}
