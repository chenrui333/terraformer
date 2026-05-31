// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	githubAPI "github.com/google/go-github/v88/github"
)

// MembersGenerator holds GithubService struct of Terraform service information
type MembersGenerator struct {
	GithubService
}

// InitResources generates TerraformResources from Github API,
func (g *MembersGenerator) InitResources() error {
	ctx := context.Background()
	client, err := g.createClient()
	if err != nil {
		return err
	}

	owner := g.Args["owner"].(string)
	resources, err := createMembershipsResources(ctx, client, owner)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, resources...)

	return nil
}

func createMembershipsResources(ctx context.Context, client *githubAPI.Client, owner string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}

	opt := &githubAPI.ListMembersOptions{
		ListOptions: githubAPI.ListOptions{PerPage: 100},
	}

	// List all organization members for the authenticated user
	for {
		members, resp, err := client.Organizations.ListMembers(ctx, owner, opt)
		if err != nil {
			return nil, fmt.Errorf("list github members for %s: %w", owner, err)
		}

		for _, member := range members {
			resource := terraformutils.NewSimpleResource(
				owner+":"+member.GetLogin(),
				member.GetLogin(),
				"github_membership",
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
