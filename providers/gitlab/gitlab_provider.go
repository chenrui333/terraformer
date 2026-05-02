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
	if len(args) < 1 || args[0] == "" {
		return errors.New("gitlab: group is required")
	}

	p.group = args[0]
	p.baseURL = gitLabDefaultURL
	if len(args) < 2 {
		if os.Getenv("GITLAB_TOKEN") == "" {
			return errors.New("token requirement")
		}
		p.token = os.Getenv("GITLAB_TOKEN")
	} else {
		p.token = args[1]
	}
	if len(args) > 2 {
		if args[2] != "" {
			p.baseURL = args[2]
		}
	}
	return nil
}

func (p *GitLabProvider) GetName() string {
	return "gitlab"
}

func (p *GitLabProvider) InitService(serviceName string, verbose bool) error {
	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
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
