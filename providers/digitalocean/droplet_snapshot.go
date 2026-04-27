// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

type DropletSnapshotGenerator struct {
	DigitalOceanService
}

func (g DropletSnapshotGenerator) listDropletSnapshots(ctx context.Context, client *godo.Client) ([]godo.Snapshot, error) {
	list := []godo.Snapshot{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		snapshots, resp, err := client.Snapshots.ListDroplet(ctx, opt)
		if err != nil {
			return nil, err
		}

		list = append(list, snapshots...)

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

func (g DropletSnapshotGenerator) createResources(snapshotList []godo.Snapshot) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, snapshot := range snapshotList {
		resources = append(resources, terraformutils.NewSimpleResource(
			snapshot.ID,
			snapshot.Name,
			"digitalocean_droplet_snapshot",
			"digitalocean",
			[]string{}))
	}
	return resources
}

func (g *DropletSnapshotGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listDropletSnapshots(context.TODO(), client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
