// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

type CDNGenerator struct {
	DigitalOceanService
}

func (g CDNGenerator) listCDNs(ctx context.Context, client *godo.Client) ([]godo.CDN, error) {
	list := []godo.CDN{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		cdns, resp, err := client.CDNs.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		list = append(list, cdns...)

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

func (g CDNGenerator) createResources(cdnList []godo.CDN) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, cdn := range cdnList {
		resources = append(resources, terraformutils.NewSimpleResource(
			cdn.ID,
			cdn.ID,
			"digitalocean_cdn",
			"digitalocean",
			[]string{}))
	}
	return resources
}

func (g *CDNGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listCDNs(context.TODO(), client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
