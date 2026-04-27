// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

type SSHKeyGenerator struct {
	DigitalOceanService
}

func (g SSHKeyGenerator) listKeys(ctx context.Context, client *godo.Client) ([]godo.Key, error) {
	list := []godo.Key{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		keys, resp, err := client.Keys.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		list = append(list, keys...)

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

func (g SSHKeyGenerator) createResources(keyList []godo.Key) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, key := range keyList {
		resources = append(resources, terraformutils.NewSimpleResource(
			strconv.Itoa(key.ID),
			key.Name,
			"digitalocean_ssh_key",
			"digitalocean",
			[]string{}))
	}
	return resources
}

func (g *SSHKeyGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listKeys(context.TODO(), client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
