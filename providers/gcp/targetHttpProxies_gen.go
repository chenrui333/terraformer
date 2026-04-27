// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var targetHttpProxiesAllowEmptyValues = []string{""}

var targetHttpProxiesAdditionalFields = map[string]interface{}{}

type TargetHttpProxiesGenerator struct {
	GCPService
}

// Run on targetHttpProxiesList and create for each TerraformResource
func (g TargetHttpProxiesGenerator) createResources(ctx context.Context, targetHttpProxiesList *compute.TargetHttpProxiesListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := targetHttpProxiesList.Pages(ctx, func(page *compute.TargetHttpProxyList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_target_http_proxy",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				targetHttpProxiesAllowEmptyValues,
				targetHttpProxiesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each targetHttpProxies create 1 TerraformResource
// Need targetHttpProxies name as ID for terraform resource
func (g *TargetHttpProxiesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	targetHttpProxiesList := computeService.TargetHttpProxies.List(g.GetArgs()["project"].(string))
	g.Resources = g.createResources(ctx, targetHttpProxiesList)

	return nil

}
