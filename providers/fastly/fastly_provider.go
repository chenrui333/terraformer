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
	if os.Getenv("FASTLY_API_KEY") == "" {
		return errors.New("set FASTLY_API_KEY env var")
	}
	p.apiKey = os.Getenv("FASTLY_API_KEY")

	if os.Getenv("FASTLY_CUSTOMER_ID") == "" {
		return errors.New("set FASTLY_CUSTOMER_ID env var")
	}
	p.customerID = os.Getenv("FASTLY_CUSTOMER_ID")

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
	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("fastly: " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"customer_id": p.customerID,
		"api_key":     p.apiKey,
	})
	return nil
}
