// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

type ProjectGenerator struct {
	DigitalOceanService
}

func (g ProjectGenerator) listProjects(ctx context.Context, client *godo.Client) ([]godo.Project, error) {
	list := []godo.Project{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		projects, resp, err := client.Projects.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		list = append(list, projects...)

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

func (g ProjectGenerator) createResources(projectList []godo.Project) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, project := range projectList {
		resources = append(resources, terraformutils.NewSimpleResource(
			project.ID,
			project.Name,
			"digitalocean_project",
			"digitalocean",
			[]string{}))
	}
	return resources
}

func (g *ProjectGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listProjects(context.TODO(), client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
