// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/zclconf/go-cty/cty"
)

type MackerelProvider struct { //nolint
	terraformutils.Provider
	apiKey         string
	mackerelClient *mackerel.Client
}

// Init check env params and initialize API Client
func (p *MackerelProvider) Init(args []string) error {
	if len(args) > 0 && args[0] != "" {
		p.apiKey = args[0]
	} else {
		if apiKey := os.Getenv("MACKEREL_API_KEY"); apiKey != "" {
			p.apiKey = apiKey
		} else {
			return errors.New("api-key requirement")
		}
	}
	// Initialize the Mackerel API client
	p.mackerelClient = mackerel.NewClient(p.apiKey)
	return nil
}

// InitService ...
func (p *MackerelProvider) InitService(serviceName string, verbose bool) error {
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
		"api-key":        p.apiKey,
		"mackerelClient": p.mackerelClient,
	})
	return nil
}

// GetName return string of provider name for Mackerel
func (p *MackerelProvider) GetName() string {
	return "mackerel"
}

// GetConfig return map of provider config for Mackerel
func (p *MackerelProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"api_key": cty.StringVal(p.apiKey),
	})
}

// GetSupportedService return map of support service for Mackerel
func (p *MackerelProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"alert_group_setting": &AlertGroupSettingGenerator{},
		"aws_integration":     &AWSIntegrationGenerator{},
		"channel":             &ChannelGenerator{},
		"downtime":            &DowntimeGenerator{},
		"monitor":             &MonitorGenerator{},
		"notification_group":  &NotificationGroupGenerator{},
		"role":                &RoleGenerator{},
		"service":             &ServiceGenerator{},
	}
}

// GetProviderData return map of provider data for Mackerel
func (p MackerelProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

// GetResourceConnections return map of resource connections for Mackerel
func (p *MackerelProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}
