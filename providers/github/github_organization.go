// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
)

type OrganizationGenerator struct {
	GithubService
}

// Generate TerraformResources from Github API
func (g *OrganizationGenerator) InitResources() error {
	ctx := context.Background()
	client, err := g.createClient()
	if err != nil {
		return err
	}

	owner := g.Args["owner"].(string)
	membershipResources, err := createMembershipsResources(ctx, client, owner)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, membershipResources...)
	blockResources, err := createOrganizationBlocksResources(ctx, client, owner)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, blockResources...)
	projectResources, err := createOrganizationProjects(ctx, client, owner)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, projectResources...)

	return nil
}
