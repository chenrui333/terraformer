// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

type CertificateGenerator struct {
	DigitalOceanService
}

func (g CertificateGenerator) listCertificates(ctx context.Context, client *godo.Client) ([]godo.Certificate, error) {
	list := []godo.Certificate{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		certificates, resp, err := client.Certificates.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		list = append(list, certificates...)

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	return list, nil
}

func (g CertificateGenerator) createResources(certificateList []godo.Certificate) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, certificate := range certificateList {
		resources = append(resources, terraformutils.NewSimpleResource(
			certificate.ID,
			certificate.Name,
			"digitalocean_certificate",
			"digitalocean",
			[]string{}))
	}
	return resources
}

func (g *CertificateGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listCertificates(context.TODO(), client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
