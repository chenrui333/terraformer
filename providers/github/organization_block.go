// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	githubAPI "github.com/google/go-github/v35/github"
)

type OrganizationBlockGenerator struct {
	GithubService
}

// Generate TerraformResources from Github API,
func (g *OrganizationBlockGenerator) InitResources() error {
	ctx := context.Background()
	client, err := g.createClient()
	if err != nil {
		return err
	}

	owner := g.Args["owner"].(string)
	resources, err := createOrganizationBlocksResources(ctx, client, owner)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, resources...)

	return nil
}

func createOrganizationBlocksResources(ctx context.Context, client *githubAPI.Client, owner string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}

	opt := &githubAPI.ListOptions{PerPage: 100}

	// List all organization blocks for the authenticated user
	for {
		blocks, resp, err := client.Organizations.ListBlockedUsers(ctx, owner, opt)
		if err != nil {
			return nil, fmt.Errorf("list github organization blocks for %s: %w", owner, err)
		}

		for _, block := range blocks {
			resource := terraformutils.NewSimpleResource(
				block.GetLogin(),
				block.GetLogin(),
				"github_organization_block",
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
	return resources, nil
}
