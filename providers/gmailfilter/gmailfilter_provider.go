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
	p.credentials = ""
	p.impersonatedUserEmail = ""

	previousCredentials, hadPreviousCredentials := os.LookupEnv("GOOGLE_CREDENTIALS")
	credentials := previousCredentials
	if len(args) > 0 && args[0] != "" {
		if err := terraformutils.SetEnv("GOOGLE_CREDENTIALS", args[0]); err != nil {
			return err
		}
		credentials = args[0]
	}
	email := os.Getenv("IMPERSONATED_USER_EMAIL")
	if len(args) > 1 && args[1] != "" {
		if err := terraformutils.SetEnv("IMPERSONATED_USER_EMAIL", args[1]); err != nil {
			if len(args) > 0 && args[0] != "" {
				if restoreErr := restoreEnv("GOOGLE_CREDENTIALS", previousCredentials, hadPreviousCredentials); restoreErr != nil {
					return errors.Join(err, restoreErr)
				}
			}
			return err
		}
		email = args[1]
	}

	p.credentials = credentials
	p.impersonatedUserEmail = email

	return nil
}

func restoreEnv(key, previousValue string, hadPreviousValue bool) error {
	if hadPreviousValue {
		return terraformutils.SetEnv(key, previousValue)
	}
	return terraformutils.UnsetEnv(key)
}

func (p *GmailfilterProvider) GetName() string {
	return "gmailfilter"
}

func (p *GmailfilterProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New("gmailfilter: " + serviceName + " not supported service")
	}
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
