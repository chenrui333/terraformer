// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/google/go-github/v35/github"
	"golang.org/x/oauth2"
)

const githubDefaultURL = "https://api.github.com/"

type GithubService struct { //nolint
	terraformutils.Service
}

func (g *GithubService) createClient() (*github.Client, error) {
	if g.GetArgs()["base_url"].(string) == githubDefaultURL {
		return g.createRegularClient()
	}
	return g.createEnterpriseClient()
}

func (g *GithubService) createRegularClient() (*github.Client, error) {
	ctx := context.Background()
	if g.Args["app_id"].(int64) != 0 && g.Args["installation_id"].(int64) != 0 && g.Args["pem"].(string) != "" {
		itr, err := ghinstallation.New(http.DefaultTransport, g.Args["app_id"].(int64), g.Args["installation_id"].(int64), []byte(g.Args["pem"].(string)))
		if err != nil {
			return nil, err
		}
		return github.NewClient(&http.Client{Transport: itr}), nil
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: g.Args["token"].(string)},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}

func (g *GithubService) createEnterpriseClient() (*github.Client, error) {
	ctx := context.Background()
	baseURL := g.GetArgs()["base_url"].(string)
	if g.Args["app_id"].(int64) != 0 && g.Args["installation_id"].(int64) != 0 && g.Args["pem"].(string) != "" {
		itr, err := ghinstallation.New(http.DefaultTransport, g.Args["app_id"].(int64), g.Args["installation_id"].(int64), []byte(g.Args["pem"].(string)))
		if err != nil {
			return nil, err
		}
		return github.NewEnterpriseClient(baseURL, baseURL, &http.Client{Transport: itr})
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: g.Args["token"].(string)},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewEnterpriseClient(baseURL, baseURL, tc)
}
