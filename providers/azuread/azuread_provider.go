// SPDX-License-Identifier: Apache-2.0

package azuread

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

type AzureADProvider struct { //nolint
	terraformutils.Provider
	tenantID     string
	clientID     string
	clientSecret string
}

func (p *AzureADProvider) setEnvConfig() error {
	p.tenantID = ""
	p.clientID = ""
	p.clientSecret = ""

	tenantID := os.Getenv("ARM_TENANT_ID")
	if tenantID == "" {
		return errors.New("please set ARM_TENANT_ID in your environment")
	}
	clientID := os.Getenv("ARM_CLIENT_ID")
	if clientID == "" {
		return errors.New("please set ARM_CLIENT_ID in your environment")
	}
	clientSecret := os.Getenv("ARM_CLIENT_SECRET")
	if clientSecret == "" {
		return errors.New("please set ARM_CLIENT_SECRET in your environment")
	}
	p.tenantID = tenantID
	p.clientID = clientID
	p.clientSecret = clientSecret
	return nil
}

func (p *AzureADProvider) Init(_ []string) error {
	err := p.setEnvConfig()
	if err != nil {
		return err
	}

	return nil
}

func (p *AzureADProvider) GetName() string {
	return "azuread"
}

func (p *AzureADProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (AzureADProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *AzureADProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"user":                &UserServiceGenerator{},
		"application":         &ApplicationServiceGenerator{},
		"group":               &GroupServiceGenerator{},
		"service_principal":   &ServicePrincipalServiceGenerator{},
		"app_role_assignment": &AppRoleAssignmentServiceGenerator{},
	}
}

func (p *AzureADProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("azuread: " + serviceName + " not supported service")
	}
	p.Service = service
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"tenant_id":     p.tenantID,
		"client_id":     p.clientID,
		"client_secret": p.clientSecret,
	})
	return nil
}
