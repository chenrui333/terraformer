// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type GitLabProvider struct { //nolint
	terraformutils.Provider
	group   string
	token   string
	baseURL string
}

func (p GitLabProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p GitLabProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"gitlab": map[string]interface{}{
				// TODO: Should I add some default config here?
				// "token": p.token,
				// "base_url": p.baseURL,
			},
		},
	}
}

func (p *GitLabProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"token": cty.StringVal(p.token),
		// NOTE: Real provider doesn't support empty/null base_url, only set when there's value
		"base_url": cty.StringVal(p.baseURL),
	})
}

// Init GitLabProvider with group
func (p *GitLabProvider) Init(args []string) error {
	p.group = ""
	p.token = ""
	p.baseURL = gitLabDefaultURL

	if len(args) < 1 || args[0] == "" {
		return errors.New("gitlab: group is required")
	}

	p.group = args[0]
	if len(args) > 1 && args[1] != "" {
		p.token = args[1]
	} else {
		token := os.Getenv("GITLAB_TOKEN")
		if token == "" {
			return errors.New("token requirement")
		}
		p.token = token
	}
	if len(args) > 2 && args[2] != "" {
		p.baseURL = args[2]
	}
	return nil
}

func (p *GitLabProvider) GetName() string {
	return "gitlab"
}

func (p *GitLabProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		"group":    p.group,
		"token":    p.token,
		"base_url": p.baseURL,
	})
	return nil
}

// GetSupportedService return map of support service for gitlab
func (p *GitLabProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"projects": &ProjectGenerator{},
		"groups":   &GroupGenerator{},
	}
}
