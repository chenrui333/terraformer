// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"log"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"

	githubAPI "github.com/google/go-github/v35/github"
)

type OrganizationWebhooksGenerator struct {
	GithubService
}

// Generate TerraformResources from Github API,
func (g *OrganizationWebhooksGenerator) InitResources() error {
	ctx := context.Background()
	client, err := g.createClient()
	if err != nil {
		return err
	}

	opt := &githubAPI.ListOptions{PerPage: 100}

	// List all organization hooks for the authenticated user
	for {
		hooks, resp, err := client.Organizations.ListHooks(ctx, g.Args["owner"].(string), opt)
		if err != nil {
			log.Println(err)
			return nil
		}

		for _, hook := range hooks {
			resource := terraformutils.NewSimpleResource(
				strconv.FormatInt(hook.GetID(), 10),
				strconv.FormatInt(hook.GetID(), 10),
				"github_organization_webhook",
				"github",
				[]string{},
			)
			resource.SlowQueryRequired = true
			g.Resources = append(g.Resources, resource)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil
}
