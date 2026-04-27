// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

type LinodeProvider struct { //nolint
	terraformutils.Provider
	token string
}

func (p *LinodeProvider) Init(_ []string) error {
	if os.Getenv("LINODE_TOKEN") == "" {
		return errors.New("set LINODE_TOKEN env var")
	}
	p.token = os.Getenv("LINODE_TOKEN")

	return nil
}

func (p *LinodeProvider) GetName() string {
	return "linode"
}

func (p *LinodeProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (LinodeProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *LinodeProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"domain":       &DomainGenerator{},
		"image":        &ImageGenerator{},
		"instance":     &InstanceGenerator{},
		"nodebalancer": &NodeBalancerGenerator{},
		"rdns":         &RDNSGenerator{},
		"sshkey":       &SSHKeyGenerator{},
		"stackscript":  &StackScriptGenerator{},
		"token":        &TokenGenerator{},
		"volume":       &VolumeGenerator{},
	}
}

func (p *LinodeProvider) InitService(serviceName string, verbose bool) error {
	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("linode: " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"token": p.token,
	})
	return nil
}
