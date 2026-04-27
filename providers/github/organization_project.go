// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"log"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"

	githubAPI "github.com/google/go-github/v35/github"
)

type OrganizationProjectGenerator struct {
	GithubService
}

// Generate TerraformResources from Github API,
func (g *OrganizationProjectGenerator) InitResources() error {
	ctx := context.Background()
	client, err := g.createClient()
	if err != nil {
		return err
	}

	owner := g.Args["owner"].(string)
	g.Resources = append(g.Resources, createOrganizationProjects(ctx, client, owner)...)

	return nil
}

func createOrganizationProjects(ctx context.Context, client *githubAPI.Client, owner string) []terraformutils.Resource {
	resources := []terraformutils.Resource{}

	opt := &githubAPI.ProjectListOptions{
		ListOptions: githubAPI.ListOptions{PerPage: 100},
	}

	// List all organization projects for the authenticated user
	for {
		projects, resp, err := client.Organizations.ListProjects(ctx, owner, opt)
		if err != nil {
			log.Println(err)
			return nil
		}

		for _, project := range projects {
			resource := terraformutils.NewSimpleResource(
				strconv.FormatInt(project.GetID(), 10),
				strconv.FormatInt(project.GetID(), 10),
				"github_organization_project",
				"github",
				[]string{},
			)
			resource.SlowQueryRequired = true
			resources = append(resources, resource)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return resources
}
