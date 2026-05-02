// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type LaunchDarklyProvider struct { //nolint
	terraformutils.Provider
	apiKey string
	client *ldapi.APIClient
	ctx    context.Context
}

const APIVersion = "20240415"

func (p *LaunchDarklyProvider) Init(_ []string) error {
	if os.Getenv("LAUNCHDARKLY_ACCESS_TOKEN") == "" {
		return errors.New("set LAUNCHDARKLY_ACCESS_TOKEN env var")
	}
	p.apiKey = os.Getenv("LAUNCHDARKLY_ACCESS_TOKEN")

	cfg := ldapi.NewConfiguration()
	cfg.AddDefaultHeader("LD-API-Version", APIVersion)

	p.client = ldapi.NewAPIClient(cfg)

	p.ctx = context.WithValue(context.Background(), ldapi.ContextAPIKeys, map[string]ldapi.APIKey{
		"ApiKey": {Key: p.apiKey},
	})
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
		"auditLogSubscription":    &AuditLogSubscriptionGenerator{},
		"customRole":              &CustomRoleGenerator{},
		"destination":             &DestinationGenerator{},
		"project":                 &ProjectGenerator{},
		"environment":             &EnvironmentGenerator{},
		"featureFlag":             &FeatureFlagsGenerator{},
		"flagTemplates":           &FlagTemplatesGenerator{},
		"flagTrigger":             &FlagTriggerGenerator{},
		"metric":                  &MetricGenerator{},
		"relayProxyConfiguration": &RelayProxyConfigurationGenerator{},
		"segment":                 &SegmentGenerator{},
		"team":                    &TeamGenerator{},
		"teamMember":              &TeamMemberGenerator{},
		"webhook":                 &WebhookGenerator{},
	}
}

func (p *LaunchDarklyProvider) InitService(serviceName string, verbose bool) error {
	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("launchdarkly: " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"api_key": p.apiKey,
		"client":  p.client,
		"ctx":     p.ctx,
	})
	return nil
}
