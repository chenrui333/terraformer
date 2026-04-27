// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"github.com/zclconf/go-cty/cty"
)

type ProviderGenerator interface {
	Init(args []string) error
	InitService(serviceName string, verbose bool) error
	GetName() string
	GetService() ServiceGenerator
	GetConfig() cty.Value
	GetBasicConfig() cty.Value
	GetSupportedService() map[string]ServiceGenerator
	GenerateFiles()
	GetProviderData(arg ...string) map[string]interface{}
	GenerateOutputPath() error
	GetResourceConnections() map[string]map[string][]string
}

type ProviderWithSource interface {
	GetSource() string
}

type Provider struct {
	Service ServiceGenerator
	Config  cty.Value
}

func (p *Provider) Init(_ []string) error {
	panic("implement me")
}

func (p *Provider) GetConfig() cty.Value {
	return p.Config
}

func (p *Provider) GetName() string {
	panic("implement me")
}

func (p *Provider) InitService(_ string) error {
	panic("implement me")
}

func (p *Provider) GenerateOutputPath() error {
	panic("implement me")
}

func (p *Provider) GenerateFiles() {
	panic("implement me")
}

func (p *Provider) GetService() ServiceGenerator {
	return p.Service
}

func (p *Provider) GetSupportedService() map[string]ServiceGenerator {
	panic("implement me")
}

func (p *Provider) GetBasicConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{})
}
