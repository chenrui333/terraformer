// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/pkg/errors"
)

type OpenStackProvider struct { //nolint
	terraformutils.Provider
	region string
}

func (p OpenStackProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p OpenStackProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"openstack": map[string]interface{}{
				"region": p.region,
			},
		},
	}
}

// check projectName in env params
func (p *OpenStackProvider) Init(args []string) error {
	p.region = ""
	if err := os.Unsetenv("OS_REGION_NAME"); err != nil {
		return err
	}

	if len(args) < 1 {
		return errors.New("openstack: expected 1 init arg (region)")
	}

	region := args[0]
	// terraform work with env param OS_REGION_NAME
	if err := os.Setenv("OS_REGION_NAME", region); err != nil {
		return err
	}
	p.region = region
	return nil
}

func (p *OpenStackProvider) GetName() string {
	return "openstack"
}

func (p *OpenStackProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New("openstack: " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		"region": p.region,
	})
	return nil
}

// GetOpenStackSupportService return map of support service for OpenStack
func (p *OpenStackProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"blockstorage": &BlockStorageGenerator{},
		"compute":      &ComputeGenerator{},
		"networking":   &NetworkingGenerator{},
	}
}
