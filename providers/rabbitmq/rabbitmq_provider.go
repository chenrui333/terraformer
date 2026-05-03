// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

type RBTProvider struct {
	terraformutils.Provider
	endpoint string
	username string
	password string
}

func (p *RBTProvider) Init(args []string) error {
	if len(args) < 3 {
		return errors.New("rabbitmq: endpoint, username, and password are required")
	}

	p.endpoint = args[0]
	p.username = args[1]
	p.password = args[2]
	return nil
}

func (p *RBTProvider) GetName() string {
	return "rabbitmq"
}

func (p *RBTProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (p *RBTProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"endpoint": cty.StringVal(p.endpoint),
		"username": cty.StringVal(p.username),
		"password": cty.StringVal(p.password),
	})
}

func (p *RBTProvider) GetBasicConfig() cty.Value {
	return p.GetConfig()
}

func (p *RBTProvider) InitService(serviceName string, verbose bool) error {
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
		"endpoint": p.endpoint,
		"username": p.username,
		"password": p.password,
	})
	return nil
}

func (p *RBTProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"bindings":    &BindingGenerator{},
		"exchanges":   &ExchangeGenerator{},
		"permissions": &PermissionsGenerator{},
		"policies":    &PolicyGenerator{},
		"queues":      &QueueGenerator{},
		"users":       &UserGenerator{},
		"vhosts":      &VhostGenerator{},
		"shovels":     &ShovelGenerator{},
	}
}

func (RBTProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{
		"bindings": {
			"exchanges": []string{"source", "name", "destination", "name"},
			"queues":    []string{"destination", "name"},
			"vhosts":    []string{"vhost", "self_link"},
		},
		"exchanges": {
			"vhosts": []string{"vhost", "self_link"},
		},
		"shovels": {
			"vhosts": []string{"vhost", "self_link"},
		},
		"permissions": {
			"users":  []string{"user", "self_link"},
			"vhosts": []string{"vhost", "self_link"},
		},
		"policies": {
			"vhosts": []string{"vhost", "self_link"},
		},
		"queues": {
			"vhosts": []string{"vhost", "self_link"},
		},
	}
}
