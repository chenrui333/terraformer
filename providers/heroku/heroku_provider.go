// SPDX-License-Identifier: Apache-2.0

package heroku

import (
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
)

type HerokuProvider struct { //nolint
	terraformutils.Provider
	apiKey string
	team   string
}

// Init the Provider for imports. args are defined in cmd/provider_cmd_heroku.go
func (p *HerokuProvider) Init(args []string) error {
	if len(args) > 0 {
		p.apiKey = args[0]
	}
	if len(args) > 1 {
		p.team = args[1]
	}
	return nil
}

func (p *HerokuProvider) GetName() string {
	return "heroku"
}

func (p *HerokuProvider) GetSource() string {
	return "heroku/heroku"
}

func (p *HerokuProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"heroku": map[string]interface{}{},
		},
	}
}

func (HerokuProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *HerokuProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"account_feature":   &AccountFeatureGenerator{},
		"app":               &AppGenerator{},
		"pipeline":          &PipelineGenerator{},
		"pipeline_coupling": &PipelineCouplingGenerator{},
		"team_collaborator": &TeamCollaboratorGenerator{},
		"team_member":       &TeamMemberGenerator{},
	}
}

func (p *HerokuProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("heroku: " + serviceName + " not supported service")
	}
	p.Service = service
	terraformutils.ConfigureService(p.Service, serviceName, verbose, p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"api_key": p.apiKey,
		"team":    p.team,
	})
	return nil
}
