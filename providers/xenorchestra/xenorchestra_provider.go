// SPDX-License-Identifier: Apache-2.0

package xenorchestra

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

type XenorchestraProvider struct { //nolint
	terraformutils.Provider
	url      string
	user     string
	password string
}

func (p *XenorchestraProvider) Init(_ []string) error {
	if os.Getenv("XOA_URL") == "" {
		return errors.New("set XOA_URL env var")
	}
	p.url = os.Getenv("XOA_URL")

	if os.Getenv("XOA_USER") == "" {
		return errors.New("set XOA_USER env var")
	}
	p.user = os.Getenv("XOA_USER")

	if os.Getenv("XOA_PASSWORD") == "" {
		return errors.New("set XOA_PASSWORD env var")
	}
	p.password = os.Getenv("XOA_PASSWORD")

	return nil
}

func (p *XenorchestraProvider) GetName() string {
	return "xenorchestra"
}

func (p *XenorchestraProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"xenorchestra": map[string]interface{}{
				"url":      p.url,
				"username": p.user,
				"password": p.password,
			},
		},
	}
}

func (XenorchestraProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *XenorchestraProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"acl":          &AclGenerator{},
		"resource_set": &ResourceSetGenerator{},
	}
}

func (p *XenorchestraProvider) InitService(serviceName string, verbose bool) error {
	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("xenorchestra: " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"url":      p.url,
		"username": p.user,
		"password": p.password,
	})
	return nil
}
