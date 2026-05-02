// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type RelayProxyConfigurationGenerator struct {
	LaunchDarklyService
}

func (g *RelayProxyConfigurationGenerator) loadRelayProxyConfigurations(ctx context.Context, client *ldapi.APIClient) error {
	configs, resp, err := client.RelayProxyConfigurationsApi.GetRelayProxyConfigs(ctx).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return err
	}
	if configs == nil {
		return nil
	}
	for _, config := range configs.Items {
		resource := terraformutils.NewResource(
			config.Id,
			resourceName(config.Name, config.Id),
			"launchdarkly_relay_proxy_configuration",
			"launchdarkly",
			map[string]string{},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *RelayProxyConfigurationGenerator) InitResources() error {
	return g.loadRelayProxyConfigurations(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient))
}
