// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

type FirewallGenerator struct {
	DigitalOceanService
}

func (g FirewallGenerator) listFirewalls(ctx context.Context, client *godo.Client) ([]godo.Firewall, error) {
	list := []godo.Firewall{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		firewalls, resp, err := client.Firewalls.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		list = append(list, firewalls...)

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

func (g FirewallGenerator) createResources(firewallList []godo.Firewall) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, firewall := range firewallList {
		resources = append(resources, terraformutils.NewSimpleResource(
			firewall.ID,
			firewall.Name,
			"digitalocean_firewall",
			"digitalocean",
			[]string{}))
	}
	return resources
}

func (g *FirewallGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listFirewalls(context.TODO(), client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
