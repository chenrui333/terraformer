// SPDX-License-Identifier: Apache-2.0

package gmailfilter

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

type GmailfilterProvider struct { //nolint
	terraformutils.Provider
	credentials           string
	impersonatedUserEmail string
}

func (p *GmailfilterProvider) Init(args []string) error {
	credentials := os.Getenv("GOOGLE_CREDENTIALS")
	if len(args) > 0 && args[0] != "" {
		credentials = args[0]
		os.Setenv("GOOGLE_CREDENTIALS", credentials)
	}
	email := os.Getenv("IMPERSONATED_USER_EMAIL")
	if len(args) > 1 && args[1] != "" {
		email = args[1]
		os.Setenv("IMPERSONATED_USER_EMAIL", email)
	}

	p.credentials = credentials
	p.impersonatedUserEmail = email

	return nil
}

func (p *GmailfilterProvider) GetName() string {
	return "gmailfilter"
}

func (p *GmailfilterProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("gmailfilter: " + serviceName + " not supported service")
	}
	p.Service = service
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"credentials":           p.credentials,
		"impersonatedUserEmail": p.impersonatedUserEmail,
	})
	return nil
}

// GetGCPSupportService return map of support service for GCP
func (p *GmailfilterProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	services := make(map[string]terraformutils.ServiceGenerator)
	services["label"] = &LabelGenerator{}
	services["filter"] = &FilterGenerator{}
	return services
}

func (p *GmailfilterProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{
		"filter": {
			"label": {
				"action.add_label_ids", "id",
			},
		},
	}
}

func (p *GmailfilterProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}
