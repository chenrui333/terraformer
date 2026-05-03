// SPDX-License-Identifier: Apache-2.0

package fastly

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

type FastlyProvider struct { //nolint
	terraformutils.Provider
	customerID string
	apiKey     string
}

func (p *FastlyProvider) Init(_ []string) error {
	p.apiKey = ""
	p.customerID = ""

	apiKey := os.Getenv("FASTLY_API_KEY")
	if apiKey == "" {
		return errors.New("set FASTLY_API_KEY env var")
	}

	customerID := os.Getenv("FASTLY_CUSTOMER_ID")
	if customerID == "" {
		return errors.New("set FASTLY_CUSTOMER_ID env var")
	}
	p.apiKey = apiKey
	p.customerID = customerID

	return nil
}

func (p *FastlyProvider) GetName() string {
	return "fastly"
}

func (p *FastlyProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"fastly": map[string]interface{}{
				"customer_id": p.customerID,
			},
		},
	}
}

func (FastlyProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *FastlyProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"service_v1":       &ServiceV1Generator{},
		"tls_subscription": &TLSSubscriptionGenerator{},
		"user":             &UserGenerator{},
	}
}

func (p *FastlyProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("fastly: " + serviceName + " not supported service")
	}
	p.Service = service
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"customer_id": p.customerID,
		"api_key":     p.apiKey,
	})
	return nil
}
