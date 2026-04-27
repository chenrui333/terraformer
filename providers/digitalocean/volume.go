// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

type VolumeGenerator struct {
	DigitalOceanService
}

func (g VolumeGenerator) listVolumes(ctx context.Context, client *godo.Client) ([]godo.Volume, error) {
	list := []godo.Volume{}

	// create options. initially, these will be blank
	opt := &godo.ListVolumeParams{}
	for {
		volumes, resp, err := client.Storage.ListVolumes(ctx, opt)
		if err != nil {
			return nil, err
		}

		list = append(list, volumes...)

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.ListOptions.Page = page + 1
	}

	return list, nil
}

func (g VolumeGenerator) createResources(volumeList []godo.Volume) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, volume := range volumeList {
		resources = append(resources, terraformutils.NewSimpleResource(
			volume.ID,
			volume.Name,
			"digitalocean_volume",
			"digitalocean",
			[]string{}))
	}
	return resources
}

func (g *VolumeGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listVolumes(context.TODO(), client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
