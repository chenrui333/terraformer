// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

type LaunchDarklyProvider struct { //nolint
	terraformutils.Provider
	apiKey string
	client *ldapi.APIClient
	ctx    context.Context
}

const APIVersion = "20240415"

func (p *LaunchDarklyProvider) Init(_ []string) error {
	p.apiKey = ""
	p.client = nil
	p.ctx = nil

	apiKey := os.Getenv("LAUNCHDARKLY_ACCESS_TOKEN")
	if apiKey == "" {
		return errors.New("set LAUNCHDARKLY_ACCESS_TOKEN env var")
	}

	cfg := ldapi.NewConfiguration()
	cfg.AddDefaultHeader("LD-API-Version", APIVersion)

	client := ldapi.NewAPIClient(cfg)

	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKeys, map[string]ldapi.APIKey{
		"ApiKey": {Key: apiKey},
	})
	p.apiKey = apiKey
	p.client = client
	p.ctx = ctx
	return nil
}

func (p *LaunchDarklyProvider) GetName() string {
	return "launchdarkly"
}

func (p *LaunchDarklyProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"launchdarkly": map[string]interface{}{
				"access_token": p.apiKey,
			},
		},
	}
}

func (LaunchDarklyProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *LaunchDarklyProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"accessToken":             &AccessTokenGenerator{},
		"aiConfig":                &AIConfigGenerator{},
		"aiConfigVariation":       &AIConfigVariationGenerator{},
		"aiTool":                  &AIToolGenerator{},
		"auditLogSubscription":    &AuditLogSubscriptionGenerator{},
		"customRole":              &CustomRoleGenerator{},
		"destination":             &DestinationGenerator{},
		"project":                 &ProjectGenerator{},
		"environment":             &EnvironmentGenerator{},
		"featureFlag":             &FeatureFlagsGenerator{},
		"flagTemplates":           &FlagTemplatesGenerator{},
		"flagTrigger":             &FlagTriggerGenerator{},
		"metric":                  &MetricGenerator{},
		"modelConfig":             &ModelConfigGenerator{},
		"relayProxyConfiguration": &RelayProxyConfigurationGenerator{},
		"segment":                 &SegmentGenerator{},
		"team":                    &TeamGenerator{},
		"teamMember":              &TeamMemberGenerator{},
		"view":                    &ViewGenerator{},
		"viewLinks":               &ViewLinksGenerator{},
		"webhook":                 &WebhookGenerator{},
	}
}

func (p *LaunchDarklyProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("launchdarkly: " + serviceName + " not supported service")
	}
	p.Service = service
	terraformutils.ConfigureService(p.Service, serviceName, verbose, p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"api_key": p.apiKey,
		"client":  p.client,
		"ctx":     p.ctx,
	})
	return nil
}
