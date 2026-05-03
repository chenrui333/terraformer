// SPDX-License-Identifier: Apache-2.0

package logzio

import (
	"regexp"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type LogzioProvider struct { //nolint
	terraformutils.Provider
	apiToken string
	baseURL  string
}

var (
	disallowedChars = regexp.MustCompile(`[^A-Za-z0-9-]`)
)

func (p LogzioProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{
		"alerts": {"alert_notification_endpoints": []string{"alert_notification_endpoints", "id"}},
	}
}

func (p LogzioProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (p *LogzioProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"api_token": cty.StringVal(p.apiToken),
		"base_url":  cty.StringVal(p.baseURL),
	})
}

// Init LogzioProvider with API apiToken
func (p *LogzioProvider) Init(args []string) error {
	p.apiToken = ""
	p.baseURL = ""

	if len(args) < 2 {
		return errors.New("logzio: api token and base URL are required")
	}
	p.apiToken = args[0]
	p.baseURL = args[1]
	return nil
}

func (p *LogzioProvider) GetName() string {
	return "logzio"
}

func (p *LogzioProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		"api_token": p.apiToken,
		"base_url":  p.baseURL,
	})
	return nil
}

// GetSupportedService return map of support service for Logzio
func (p *LogzioProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"alerts":                       &AlertsGenerator{},
		"alert_notification_endpoints": &AlertNotificationEndpointsGenerator{},
	}
}

func createSlug(s string) string {
	s = strings.ToLower(s)

	return disallowedChars.ReplaceAllString(s, "-")
}
