// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var regionTargetHttpProxiesAllowEmptyValues = []string{""}

var regionTargetHttpProxiesAdditionalFields = map[string]interface{}{}

type RegionTargetHttpProxiesGenerator struct {
	GCPService
}

// Run on regionTargetHttpProxiesList and create for each TerraformResource
func (g RegionTargetHttpProxiesGenerator) createResources(ctx context.Context, regionTargetHttpProxiesList *compute.RegionTargetHttpProxiesListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := regionTargetHttpProxiesList.Pages(ctx, func(page *compute.TargetHttpProxyList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_region_target_http_proxy",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				regionTargetHttpProxiesAllowEmptyValues,
				regionTargetHttpProxiesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each regionTargetHttpProxies create 1 TerraformResource
// Need regionTargetHttpProxies name as ID for terraform resource
func (g *RegionTargetHttpProxiesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	regionTargetHttpProxiesList := computeService.RegionTargetHttpProxies.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	g.Resources = g.createResources(ctx, regionTargetHttpProxiesList)

	return nil

}
