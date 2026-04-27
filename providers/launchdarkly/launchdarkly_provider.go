// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	launchdarkly "github.com/launchdarkly/api-client-go"
)

type LaunchDarklyProvider struct { //nolint
	terraformutils.Provider
	apiKey string
	client *launchdarkly.APIClient
	ctx    context.Context
}

const (
	basePath   = "https://app.launchdarkly.com/api/v2"
	version    = "0.0.1"
	APIVersion = "20191212"
)

func (p *LaunchDarklyProvider) Init(_ []string) error {
	if os.Getenv("LAUNCHDARKLY_ACCESS_TOKEN") == "" {
		return errors.New("set LAUNCHDARKLY_ACCESS_TOKEN env var")
	}
	p.apiKey = os.Getenv("LAUNCHDARKLY_ACCESS_TOKEN")

	cfg := &launchdarkly.Configuration{
		BasePath:      basePath,
		DefaultHeader: make(map[string]string),
		UserAgent:     fmt.Sprintf("launchdarkly-terraformer/%s", version),
	}
	cfg.AddDefaultHeader("LD-API-Version", APIVersion)

	p.client = launchdarkly.NewAPIClient(cfg)

	p.ctx = context.WithValue(context.Background(), launchdarkly.ContextAPIKey, launchdarkly.APIKey{
		Key: p.apiKey,
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
		"project":     &ProjectGenerator{},
		"featureFlag": &FeatureFlagsGenerator{},
		"segment":     &SegmentGenerator{},
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
