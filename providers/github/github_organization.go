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
	g.Resources = append(g.Resources, createMembershipsResources(ctx, client, owner)...)
	g.Resources = append(g.Resources, createOrganizationBlocksResources(ctx, client, owner)...)
	g.Resources = append(g.Resources, createOrganizationProjects(ctx, client, owner)...)

	return nil
}
