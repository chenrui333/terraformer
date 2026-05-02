// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var interconnectAttachmentsAllowEmptyValues = []string{""}

var interconnectAttachmentsAdditionalFields = map[string]interface{}{}

type InterconnectAttachmentsGenerator struct {
	GCPService
}

// Run on interconnectAttachmentsList and create for each TerraformResource
func (g InterconnectAttachmentsGenerator) createResources(ctx context.Context, interconnectAttachmentsList *compute.InterconnectAttachmentsListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := interconnectAttachmentsList.Pages(ctx, func(page *compute.InterconnectAttachmentList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_interconnect_attachment",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				interconnectAttachmentsAllowEmptyValues,
				interconnectAttachmentsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list interconnectAttachments: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each interconnectAttachments create 1 TerraformResource
// Need interconnectAttachments name as ID for terraform resource
func (g *InterconnectAttachmentsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	interconnectAttachmentsList := computeService.InterconnectAttachments.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, interconnectAttachmentsList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
