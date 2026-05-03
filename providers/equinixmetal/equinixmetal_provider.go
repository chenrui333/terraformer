// SPDX-License-Identifier: Apache-2.0

package equinixmetal

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

type EquinixMetalProvider struct { //nolint
	terraformutils.Provider
	authToken string
	projectID string
}

func (p *EquinixMetalProvider) Init(_ []string) error {
	p.authToken = ""
	p.projectID = ""

	authToken := os.Getenv("PACKET_AUTH_TOKEN")
	if authToken == "" {
		return errors.New("set PACKET_AUTH_TOKEN env var")
	}

	projectID := os.Getenv("METAL_PROJECT_ID")
	if projectID == "" {
		return errors.New("set METAL_PROJECT_ID env var")
	}
	p.authToken = authToken
	p.projectID = projectID

	return nil
}

func (p *EquinixMetalProvider) GetName() string {
	return "metal"
}

func (p *EquinixMetalProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (EquinixMetalProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *EquinixMetalProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"device":            &DeviceGenerator{},
		"sshkey":            &SSHKeyGenerator{},
		"spotmarketrequest": &SpotMarketRequestGenerator{},
		"volume":            &VolumeGenerator{},
	}
}

func (p *EquinixMetalProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("equinixmetal: " + serviceName + " not supported service")
	}
	p.Service = service
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"auth_token": p.authToken,
		"project_id": p.projectID,
	})
	return nil
}
