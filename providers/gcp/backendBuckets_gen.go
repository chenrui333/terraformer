// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var backendBucketsAllowEmptyValues = []string{""}

var backendBucketsAdditionalFields = map[string]interface{}{}

type BackendBucketsGenerator struct {
	GCPService
}

// Run on backendBucketsList and create for each TerraformResource
func (g BackendBucketsGenerator) createResources(ctx context.Context, backendBucketsList *compute.BackendBucketsListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := backendBucketsList.Pages(ctx, func(page *compute.BackendBucketList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_backend_bucket",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				backendBucketsAllowEmptyValues,
				backendBucketsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list backendBuckets: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each backendBuckets create 1 TerraformResource
// Need backendBuckets name as ID for terraform resource
func (g *BackendBucketsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	backendBucketsList := computeService.BackendBuckets.List(g.GetArgs()["project"].(string))
	resources, err := g.createResources(ctx, backendBucketsList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
