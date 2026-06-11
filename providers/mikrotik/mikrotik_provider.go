// SPDX-License-Identifier: Apache-2.0

package mikrotik

import (
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/ddelnano/terraform-provider-mikrotik/client"
)

type MikrotikProvider struct { //nolint
	terraformutils.Provider
	client.Mikrotik
}

func (p *MikrotikProvider) Init(_ []string) error {
	// The mikrotik provider gets its credentials through environment variables
	// and therefore nothing needs to be done here
	return nil
}

func (p *MikrotikProvider) GetName() string {
	return "mikrotik"
}

func (p *MikrotikProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (MikrotikProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *MikrotikProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"dhcp_lease": &DhcpLeaseGenerator{},
	}
}

func (p *MikrotikProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New("mikrotik: " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		"host":           p.Host,
		"user":           p.Username,
		"password":       p.Password,
		"tls":            p.TLS,
		"ca_certificate": p.CA,
		"insecure":       p.Insecure,
	})
	return nil
}
