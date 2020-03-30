// Copyright 2019 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pagerduty

import (
	"errors"
	"os"

	"github.com/GoogleCloudPlatform/terraformer/terraform_utils"
	"github.com/GoogleCloudPlatform/terraformer/terraform_utils/provider_wrapper"
)

type PagerDutyProvider struct {
	terraform_utils.Provider
	apiKey string
}

func (p *PagerDutyProvider) Init(args []string) error {
	if os.Getenv("PAGERDUTY_TOKEN") == "" {
		return errors.New("set PAGERDUTY_TOKEN env var")
	}
	p.apiKey = os.Getenv("PAGERDUTY_TOKEN")

	return nil
}

func (p *PagerDutyProvider) GetName() string {
	return "pagerduty"
}

func (p *PagerDutyProvider) GetProviderData(arg ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"pagerduty": map[string]interface{}{
				"version": provider_wrapper.GetProviderVersion(p.GetName()),
				"api_key": p.apiKey,
			},
		},
	}
}

func (PagerDutyProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *PagerDutyProvider) GetSupportedService() map[string]terraform_utils.ServiceGenerator {
	return map[string]terraform_utils.ServiceGenerator{
		"teams": &TeamsGenerator{},
		"users": &UsersGenerator{},
	}
}

func (p *PagerDutyProvider) InitService(serviceName string, verbose bool) error {
	var isSupported bool

	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("pagerduty: " + serviceName + " not supported service")
	}

	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"api_key": p.apiKey,
	})

	return nil
}
