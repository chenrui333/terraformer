// SPDX-License-Identifier: Apache-2.0

package pagerduty

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

type PagerDutyProvider struct { //nolint
	terraformutils.Provider
	token string
}

func (p *PagerDutyProvider) Init(args []string) error {
	if token := os.Getenv("PAGERDUTY_TOKEN"); token != "" {
		p.token = os.Getenv("PAGERDUTY_TOKEN")
	}
	if len(args) > 0 && args[0] != "" {
		p.token = args[0]
	}
	return nil
}

func (p *PagerDutyProvider) GetName() string {
	return "pagerduty"
}

func (p *PagerDutyProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"token": cty.StringVal(p.token),
	})
}

func (p *PagerDutyProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"pagerduty": map[string]interface{}{
				"token": p.token,
			},
		},
	}
}

func (PagerDutyProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *PagerDutyProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"business_service":  &BusinessServiceGenerator{},
		"escalation_policy": &EscalationPolicyGenerator{},
		"ruleset":           &RulesetGenerator{},
		"schedule":          &ScheduleGenerator{},
		"service":           &ServiceGenerator{},
		"team":              &TeamGenerator{},
		"user":              &UserGenerator{},
	}
}

func (p *PagerDutyProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service = service
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"token": p.token,
	})
	return nil
}
