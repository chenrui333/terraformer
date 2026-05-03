// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/pkg/errors"
)

type CommercetoolsProvider struct { //nolint
	terraformutils.Provider
	clientID     string
	clientSecret string
	clientScope  string
	projectKey   string
	baseURL      string
	tokenURL     string
}

func (p CommercetoolsProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p CommercetoolsProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

// Init CommerectoolsProvider
func (p *CommercetoolsProvider) Init(args []string) error {
	if len(args) < 6 {
		return errors.New("commercetools: client id, client scope, client secret, project key, base URL, and token URL are required")
	}

	p.clientID = args[0]
	p.clientScope = args[1]
	p.clientSecret = args[2]
	p.projectKey = args[3]
	p.baseURL = args[4]
	p.tokenURL = args[5]
	return nil
}

func (p *CommercetoolsProvider) GetName() string {
	return "commercetools"
}

func (p *CommercetoolsProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"client_id":     p.clientID,
		"client_secret": p.clientSecret,
		"client_scope":  p.clientScope,
		"project_key":   p.projectKey,
		"base_url":      p.baseURL,
		"token_url":     p.tokenURL,
	})
	return nil
}

// GetSupportedService return map of support service for Logzio
func (p *CommercetoolsProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"api_extension":   &APIExtensionGenerator{},
		"channel":         &ChannelGenerator{},
		"custom_object":   &CustomObjectGenerator{},
		"product_type":    &ProductTypeGenerator{},
		"shipping_zone":   &ShippingZoneGenerator{},
		"shipping_method": &ShippingMethodGenerator{},
		"state":           &StateGenerator{},
		"store":           &StoreGenerator{},
		"subscription":    &SubscriptionGenerator{},
		"tax_category":    &TaxCategoryGenerator{},
		"types":           &TypesGenerator{},
	}
}
