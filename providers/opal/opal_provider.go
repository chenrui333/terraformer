// SPDX-License-Identifier: Apache-2.0

package opal

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

const opalDefaultURL = "https://api.opal.dev"

type OpalProvider struct { //nolint
	terraformutils.Provider
	token   string
	baseURL string
}

func (p OpalProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"opal": map[string]interface{}{
				"base_url": p.baseURL,
			},
		},
	}
}

func (p *OpalProvider) GetName() string {
	return "opal"
}

func (p *OpalProvider) GetSource() string {
	return "opalsecurity/opal"
}

func (p OpalProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{
		"resource": {
			"owner": {
				"admin_owner_id", "id",
				"reviewer_stage.reviewer.id", "id",
			},
			"group": {"visibility_group.id", "id"},
		},
		"group": {
			"owner": {
				"admin_owner_id", "id",
				"reviewer_stage.reviewer.id", "id",
			},
			"group": {"visibility_group.id", "id"},
			"message_channel": {
				"audit_message_channel.id", "id",
			},
			"on_call_schedule": {
				"on_call_schedule.id", "id",
			},
		},
		"owner": {
			"message_channel": {
				"reviewer_message_channel_id", "id",
			},
		},
	}
}

func (p *OpalProvider) Init(_ []string) error {
	p.token = ""
	p.baseURL = ""

	token := os.Getenv("OPAL_AUTH_TOKEN")
	if token == "" {
		return errors.New("the Opal API key must be set via `OPAL_AUTH_TOKEN` env var")
	}
	baseURL := os.Getenv("OPAL_BASE_URL")
	if baseURL == "" {
		baseURL = opalDefaultURL
	}
	p.token = token
	p.baseURL = baseURL

	return nil
}

func (p *OpalProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"token":    cty.StringVal(p.token),
		"base_url": cty.StringVal(p.baseURL),
	})
}

func (p *OpalProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("opal: " + serviceName + " is not a supported resource type")
	}
	p.Service = service
	terraformutils.ConfigureService(p.Service, serviceName, verbose, p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"token":    p.token,
		"base_url": p.baseURL,
	})
	return nil
}

func (p *OpalProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"owner":            &OwnerGenerator{},
		"resource":         &ResourceGenerator{},
		"group":            &GroupGenerator{},
		"message_channel":  &MessageChannelGenerator{},
		"on_call_schedule": &OnCallScheduleGenerator{},
	}
}
