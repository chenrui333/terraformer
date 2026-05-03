// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

type VultrProvider struct { //nolint
	terraformutils.Provider
	apiKey string
}

func (p *VultrProvider) Init(_ []string) error {
	p.apiKey = ""

	apiKey := os.Getenv("VULTR_API_KEY")
	if apiKey == "" {
		return errors.New("set VULTR_API_KEY env var")
	}
	p.apiKey = apiKey

	return nil
}

func (p *VultrProvider) GetName() string {
	return "vultr"
}

func (p *VultrProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (VultrProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *VultrProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"bare_metal_server": &BareMetalServerGenerator{},
		"block_storage":     &BlockStorageGenerator{},
		"dns_domain":        &DNSDomainGenerator{},
		"firewall_group":    &FirewallGroupGenerator{},
		"network":           &NetworkGenerator{},
		"reserved_ip":       &ReservedIPGenerator{},
		"server":            &ServerGenerator{},
		"snapshot":          &SnapshotGenerator{},
		"ssh_key":           &SSHKeyGenerator{},
		"startup_script":    &StartupScriptGenerator{},
		"user":              &UserGenerator{},
	}
}

func (p *VultrProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New("vultr: " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		"api_key": p.apiKey,
	})
	return nil
}
