// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"errors"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

type Provider struct {
	terraformutils.Provider
}

func (p *Provider) Init(_ []string) error {
	return nil
}

func (p *Provider) GetName() string {
	return "helm"
}

func (p *Provider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (Provider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *Provider) GetConfig() cty.Value {
	return cty.EmptyObjectVal
}

func (p *Provider) GetBasicConfig() cty.Value {
	return p.GetConfig()
}

func (p *Provider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return fmt.Errorf("helm: %s not supported service", serviceName)
	}
	return nil
}

func (p *Provider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"release": &ReleaseGenerator{},
	}
}

func (p *Provider) ValidateImport(resources []string) error {
	for _, resource := range resources {
		if resource == "release" {
			return ErrReleaseImportNotImplemented
		}
		if _, ok := p.GetSupportedService()[resource]; !ok {
			return fmt.Errorf("helm: %s not supported service", resource)
		}
	}
	return nil
}

var ErrReleaseImportNotImplemented = errors.New("helm: release import is not implemented yet")
