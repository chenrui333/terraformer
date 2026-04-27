// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

type TagGenerator struct {
	DigitalOceanService
}

func (g TagGenerator) listTags(ctx context.Context, client *godo.Client) ([]godo.Tag, error) {
	list := []godo.Tag{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		tags, resp, err := client.Tags.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		list = append(list, tags...)

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

func (g TagGenerator) createResources(tagList []godo.Tag) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, tag := range tagList {
		resources = append(resources, terraformutils.NewSimpleResource(
			tag.Name,
			tag.Name,
			"digitalocean_tag",
			"digitalocean",
			[]string{}))
	}
	return resources
}

func (g *TagGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listTags(context.TODO(), client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
