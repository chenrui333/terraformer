// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var sslCertificatesAllowEmptyValues = []string{""}

var sslCertificatesAdditionalFields = map[string]interface{}{}

type SslCertificatesGenerator struct {
	GCPService
}

// Run on sslCertificatesList and create for each TerraformResource
func (g SslCertificatesGenerator) createResources(ctx context.Context, sslCertificatesList *compute.SslCertificatesListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := sslCertificatesList.Pages(ctx, func(page *compute.SslCertificateList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				obj.Name,
				"google_compute_managed_ssl_certificate",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				sslCertificatesAllowEmptyValues,
				sslCertificatesAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each sslCertificates create 1 TerraformResource
// Need sslCertificates name as ID for terraform resource
func (g *SslCertificatesGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	sslCertificatesList := computeService.SslCertificates.List(g.GetArgs()["project"].(string))
	g.Resources = g.createResources(ctx, sslCertificatesList)

	return nil

}
