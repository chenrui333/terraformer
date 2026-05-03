// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"errors"
	"os"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

type Auth0Provider struct { //nolint
	terraformutils.Provider
	domain       string
	clientID     string
	clientSecret string
	client       *management.Management
}

func (p *Auth0Provider) Init(_ []string) error {
	p.domain = ""
	p.clientID = ""
	p.clientSecret = ""
	p.client = nil

	domain := os.Getenv("AUTH0_DOMAIN")
	if domain == "" {
		return errors.New("set AUTH0_DOMAIN env var")
	}

	clientID := os.Getenv("AUTH0_CLIENT_ID")
	if clientID == "" {
		return errors.New("set AUTH0_CLIENT_ID env var")
	}

	clientSecret := os.Getenv("AUTH0_CLIENT_SECRET")
	if clientSecret == "" {
		return errors.New("set AUTH0_CLIENT_SECRET env var")
	}

	client, err := newManagementClient(domain, clientID, clientSecret)
	if err != nil {
		return err
	}
	p.domain = domain
	p.clientID = clientID
	p.clientSecret = clientSecret
	p.client = client

	return nil
}

func (p *Auth0Provider) GetName() string {
	return "auth0"
}

func (p *Auth0Provider) GetSource() string {
	return "auth0/auth0"
}

func (p *Auth0Provider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"domain":        cty.StringVal(p.domain),
		"client_id":     cty.StringVal(p.clientID),
		"client_secret": cty.StringVal(p.clientSecret),
	})
}

func (p *Auth0Provider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service = service
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"domain":            p.domain,
		"client_id":         p.clientID,
		"client_secret":     p.clientSecret,
		managementClientArg: p.client,
	})
	return nil
}

func (p *Auth0Provider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"auth0_action":          &ActionGenerator{},
		"auth0_client":          &ClientGenerator{},
		"auth0_client_grant":    &ClientGrantGenerator{},
		"auth0_hook":            &HookGenerator{},
		"auth0_resource_server": &ResourceServerGenerator{},
		"auth0_role":            &RoleGenerator{},
		"auth0_rule":            &RuleGenerator{},
		"auth0_rule_config":     &RuleConfigGenerator{},
		"auth0_trigger_binding": &TriggerBindingGenerator{},
		"auth0_user":            &UserGenerator{},
		"auth0_branding":        &BrandingGenerator{},
		"auth0_custom_domain":   &CustomDomainGenerator{},
		"auth0_email":           &EmailGenerator{},
		"auth0_prompt":          &PromptGenerator{},
		"auth0_log_stream":      &LogStreamGenerator{},
		"auth0_tenant":          &TenantGenerator{},
	}
}

func (p Auth0Provider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p Auth0Provider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}
