// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"

	githubAPI "github.com/google/go-github/v35/github"
)

type UserSSHKeyGenerator struct {
	GithubService
}

// Generate TerraformResources from Github API,
func (g *UserSSHKeyGenerator) InitResources() error {
	ctx := context.Background()
	client, err := g.createClient()
	if err != nil {
		return err
	}

	opt := &githubAPI.ListOptions{PerPage: 100}

	// List all ssh keys for the authenticated user
	for {
		keys, resp, err := client.Users.ListKeys(ctx, "", opt)
		if err != nil {
			return fmt.Errorf("list github user ssh keys: %w", err)
		}

		for _, key := range keys {
			resource := terraformutils.NewSimpleResource(
				strconv.FormatInt(key.GetID(), 10),
				strconv.FormatInt(key.GetID(), 10),
				"github_user_ssh_key",
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
