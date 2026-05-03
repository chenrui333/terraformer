//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package myrasec

import (
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
)

// MyrasecProvider
type MyrasecProvider struct {
	terraformutils.Provider
}

// Init
func (p *MyrasecProvider) Init(_ []string) error {
	return nil
}

// GetName
func (p *MyrasecProvider) GetName() string {
	return "myrasec"
}

// GetProviderData
func (p *MyrasecProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

// GetResourceConnections
func (MyrasecProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

// GetSupportedService
func (p *MyrasecProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"domain":        &DomainGenerator{},
		"dns_record":    &DNSGenerator{},
		"cache_setting": &CacheSettingGenerator{},
		"redirect":      &RedirectGenerator{},
		"ip_filter":     &IPFilterGenerator{},
		"settings":      &SettingsGenerator{},
		"waf_rule":      &WafRuleGenerator{},
		"maintenance":   &MaintenanceGenerator{},
		"error_page":    &ErrorPageGenerator{},
	}
}

// InitService
func (p *MyrasecProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("myrasec: " + serviceName + " not supported service")
	}
	p.Service = service
	terraformutils.ConfigureService(p.Service, serviceName, verbose, p.GetName())

	return nil
}
