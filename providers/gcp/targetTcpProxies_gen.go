// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var targetTcpProxiesAllowEmptyValues = []string{""}

var targetTcpProxiesAdditionalFields = map[string]interface{}{}

type TargetTcpProxiesGenerator struct {
	GCPService
}

// Run on targetTcpProxiesList and create for each TerraformResource
func (g TargetTcpProxiesGenerator) createResources(ctx context.Context, targetTcpProxiesList *compute.TargetTcpProxiesListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := targetTcpProxiesList.Pages(ctx, func(page *compute.TargetTcpProxyList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_target_tcp_proxy",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				targetTcpProxiesAllowEmptyValues,
				targetTcpProxiesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each targetTcpProxies create 1 TerraformResource
// Need targetTcpProxies name as ID for terraform resource
func (g *TargetTcpProxiesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	targetTcpProxiesList := computeService.TargetTcpProxies.List(g.GetArgs()["project"].(string))
	g.Resources = g.createResources(ctx, targetTcpProxiesList)

	return nil

}
