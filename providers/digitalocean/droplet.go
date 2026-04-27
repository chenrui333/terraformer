// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

type DropletGenerator struct {
	DigitalOceanService
}

func (g DropletGenerator) listDroplets(ctx context.Context, client *godo.Client) ([]godo.Droplet, error) {
	list := []godo.Droplet{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		list = append(list, droplets...)

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

func (g DropletGenerator) createResources(dropletList []godo.Droplet) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, droplet := range dropletList {
		resources = append(resources, terraformutils.NewSimpleResource(
			strconv.Itoa(droplet.ID),
			droplet.Name,
			"digitalocean_droplet",
			"digitalocean",
			[]string{}))
	}
	return resources
}

func (g *DropletGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listDroplets(context.TODO(), client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
