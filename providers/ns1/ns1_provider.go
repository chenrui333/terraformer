// SPDX-License-Identifier: Apache-2.0

package ns1

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

type Ns1Provider struct { //nolint
	terraformutils.Provider
	apiKey string
}

func (p *Ns1Provider) Init(_ []string) error {
	p.apiKey = ""

	apiKey := os.Getenv("NS1_APIKEY")
	if apiKey == "" {
		return errors.New("set NS1_APIKEY env var")
	}
	p.apiKey = apiKey

	return nil
}

func (p *Ns1Provider) GetName() string {
	return "ns1"
}

func (p *Ns1Provider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (Ns1Provider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *Ns1Provider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"monitoringjob": &MonitoringJobGenerator{},
		"team":          &TeamGenerator{},
		"zone":          &ZoneGenerator{},
	}
}

func (p *Ns1Provider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("ns1: " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"api_key": p.apiKey,
	})
	return nil
}
