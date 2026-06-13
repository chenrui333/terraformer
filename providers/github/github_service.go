// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/google/go-github/v88/github"
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
		return github.NewClient(github.WithHTTPClient(&http.Client{Transport: itr}))
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: g.Args["token"].(string)},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(github.WithHTTPClient(tc))
}

func (g *GithubService) createEnterpriseClient() (*github.Client, error) {
	ctx := context.Background()
	baseURL, err := normalizeGithubBaseURL(g.GetArgs()["base_url"].(string))
	if err != nil {
		return nil, err
	}
	if g.Args["app_id"].(int64) != 0 && g.Args["installation_id"].(int64) != 0 && g.Args["pem"].(string) != "" {
		itr, err := ghinstallation.New(http.DefaultTransport, g.Args["app_id"].(int64), g.Args["installation_id"].(int64), []byte(g.Args["pem"].(string)))
		if err != nil {
			return nil, err
		}
		return github.NewClient(
			github.WithEnterpriseURLs(baseURL, baseURL),
			github.WithHTTPClient(&http.Client{Transport: itr}),
		)
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: g.Args["token"].(string)},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(
		github.WithEnterpriseURLs(baseURL, baseURL),
		github.WithHTTPClient(tc),
	)
}

func normalizeGithubBaseURL(baseURL string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("github: invalid base_url %q: %w", baseURL, err)
	}
	if parsed.User != nil {
		return "", errors.New("github: invalid base_url: credentials are not allowed")
	}
	if parsed.RawQuery != "" {
		return "", errors.New("github: invalid base_url: query parameters are not allowed")
	}
	if parsed.Fragment != "" {
		return "", errors.New("github: invalid base_url: fragments are not allowed")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("github: invalid base_url %q: scheme must be http or https", baseURL)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("github: invalid base_url %q: host is required", baseURL)
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	return parsed.String(), nil
}
