// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var regionSslCertificatesAllowEmptyValues = []string{""}

var regionSslCertificatesAdditionalFields = map[string]interface{}{}

type RegionSslCertificatesGenerator struct {
	GCPService
}

// Run on regionSslCertificatesList and create for each TerraformResource
func (g RegionSslCertificatesGenerator) createResources(ctx context.Context, regionSslCertificatesList *compute.RegionSslCertificatesListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := regionSslCertificatesList.Pages(ctx, func(page *compute.SslCertificateList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_region_ssl_certificate",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				regionSslCertificatesAllowEmptyValues,
				regionSslCertificatesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list regionSslCertificates: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each regionSslCertificates create 1 TerraformResource
// Need regionSslCertificates name as ID for terraform resource
func (g *RegionSslCertificatesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	regionSslCertificatesList := computeService.RegionSslCertificates.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createResources(ctx, regionSslCertificatesList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil

}
