// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
)

type CloudflareProvider struct { //nolint
	terraformutils.Provider
}

func (p *CloudflareProvider) Init(_ []string) error {
	return nil
}

func (p *CloudflareProvider) GetName() string {
	return "cloudflare"
}

func (p *CloudflareProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (CloudflareProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *CloudflareProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"access":         &AccessGenerator{},
		"dns":            &DNSGenerator{},
		"firewall":       &FirewallGenerator{},
		"page_rule":      &PageRulesGenerator{},
		"account_member": &AccountMemberGenerator{},
	}
}

func (p *CloudflareProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("cloudflare: " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())

	return nil
}
