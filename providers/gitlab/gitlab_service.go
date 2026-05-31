// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"github.com/chenrui333/terraformer/terraformutils"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

const gitLabDefaultURL = "https://gitlab.com/api/v4/"

type GitLabService struct { //nolint
	terraformutils.Service
}

func (g *GitLabService) createClient() (*gitlab.Client, error) {
	if g.GetArgs()["base_url"].(string) == gitLabDefaultURL {
		return g.createRegularClient()
	}
	return g.createEnterpriseClient()
}

func (g *GitLabService) createRegularClient() (*gitlab.Client, error) {
	return gitlab.NewClient(g.Args["token"].(string))
}

func (g *GitLabService) createEnterpriseClient() (*gitlab.Client, error) {
	return gitlab.NewClient(g.Args["token"].(string), gitlab.WithBaseURL(g.GetArgs()["base_url"].(string)))
}
