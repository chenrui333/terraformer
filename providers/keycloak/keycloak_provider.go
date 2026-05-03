// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

const keycloakInitArgCount = 10

type KeycloakProvider struct { //nolint
	terraformutils.Provider
	url                   string
	basePath              string
	clientID              string
	clientSecret          string
	realm                 string
	clientTimeout         int
	caCert                string
	tlsInsecureSkipVerify bool
	redHatSSO             bool
	target                string
}

func getArg(arg string) string {
	if arg == "-" {
		return ""
	}
	return arg
}

func (p *KeycloakProvider) Init(args []string) error {
	if len(args) < keycloakInitArgCount {
		return fmt.Errorf("keycloak: expected %d init args, got %d", keycloakInitArgCount, len(args))
	}

	clientTimeout, err := strconv.Atoi(args[5])
	if err != nil {
		return fmt.Errorf("keycloak: invalid client timeout %q: %w", args[5], err)
	}
	tlsInsecureSkipVerify, err := strconv.ParseBool(args[7])
	if err != nil {
		return fmt.Errorf("keycloak: invalid tls insecure skip verify %q: %w", args[7], err)
	}
	redHatSSO, err := strconv.ParseBool(args[8])
	if err != nil {
		return fmt.Errorf("keycloak: invalid red hat sso %q: %w", args[8], err)
	}

	p.url = args[0]
	p.basePath = args[1]
	p.clientID = args[2]
	p.clientSecret = args[3]
	p.realm = args[4]
	p.clientTimeout = clientTimeout
	p.caCert = getArg(args[6])
	p.tlsInsecureSkipVerify = tlsInsecureSkipVerify
	p.redHatSSO = redHatSSO
	p.target = getArg(args[9])
	return nil
}

func (p *KeycloakProvider) GetName() string {
	return "keycloak"
}

func (p *KeycloakProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (p *KeycloakProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"url":                      cty.StringVal(p.url),
		"base_path":                cty.StringVal(p.basePath),
		"client_id":                cty.StringVal(p.clientID),
		"client_secret":            cty.StringVal(p.clientSecret),
		"realm":                    cty.StringVal(p.realm),
		"client_timeout":           cty.NumberIntVal(int64(p.clientTimeout)),
		"root_ca_certificate":      cty.StringVal(p.caCert),
		"tls_insecure_skip_verify": cty.BoolVal(p.tlsInsecureSkipVerify),
		"red_hat_sso":              cty.BoolVal(p.redHatSSO),
	})
}

func (p *KeycloakProvider) GetBasicConfig() cty.Value {
	return p.GetConfig()
}

func (p *KeycloakProvider) InitService(serviceName string, verbose bool) error {
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
		"url":                      p.url,
		"base_path":                p.basePath,
		"client_id":                p.clientID,
		"client_secret":            p.clientSecret,
		"realm":                    p.realm,
		"client_timeout":           p.clientTimeout,
		"root_ca_certificate":      p.caCert,
		"tls_insecure_skip_verify": p.tlsInsecureSkipVerify,
		"red_hat_sso":              p.redHatSSO,
		"target":                   p.target,
	})
	return nil
}

func (p *KeycloakProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"realms": &RealmGenerator{},
	}
}

func (KeycloakProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}
