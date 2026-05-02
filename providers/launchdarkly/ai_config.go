// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

type AIConfigGenerator struct {
	LaunchDarklyService
}

type AIConfigVariationGenerator struct {
	LaunchDarklyService
}

func getAIConfigs(ctx context.Context, client *ldapi.APIClient, projectKey string) ([]ldapi.AIConfig, error) {
	var allConfigs []ldapi.AIConfig
	for offset := int32(0); ; offset += pageSize {
		configs, resp, err := client.AIConfigsApi.GetAIConfigs(ctx, projectKey).
			Limit(pageSize).
			Offset(offset).
			Execute()
		closeResponseBody(resp)
		if err != nil {
			return nil, err
		}
		if configs == nil {
			break
		}
		allConfigs = append(allConfigs, configs.GetItems()...)
		if int64(len(allConfigs)) >= int64(configs.GetTotalCount()) {
			break
		}
	}
	return allConfigs, nil
}

func (g *AIConfigGenerator) loadAIConfigs(ctx context.Context, client *ldapi.APIClient, projectKey string) error {
	configs, err := getAIConfigs(ctx, client, projectKey)
	if err != nil {
		return err
	}
	for _, config := range configs {
		configKey := config.GetKey()
		resource := terraformutils.NewResource(
			fmt.Sprintf("%s/%s", projectKey, configKey),
			launchDarklyProjectResourceName(projectKey, config.GetName(), configKey),
			"launchdarkly_ai_config",
			"launchdarkly",
			map[string]string{
				"project_key": projectKey,
				"key":         configKey,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *AIConfigGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadAIConfigs(ctx, client, project.Key); err != nil {
			return err
		}
	}
	return nil
}

func (g *AIConfigVariationGenerator) loadAIConfigVariations(ctx context.Context, client *ldapi.APIClient, projectKey string) error {
	configs, err := getAIConfigs(ctx, client, projectKey)
	if err != nil {
		return err
	}
	for _, config := range configs {
		configKey := config.GetKey()
		for _, variation := range config.GetVariations() {
			variationKey := variation.GetKey()
			resource := terraformutils.NewResource(
				fmt.Sprintf("%s/%s/%s", projectKey, configKey, variationKey),
				fmt.Sprintf("%s-%s", projectKey, resourceNameWithID(variation.GetName(), variationKey)),
				"launchdarkly_ai_config_variation",
				"launchdarkly",
				aiConfigVariationAttributes(projectKey, configKey, variationKey, variation.GetTools()),
				[]string{},
				map[string]interface{}{})
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *AIConfigVariationGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadAIConfigVariations(ctx, client, project.Key); err != nil {
			return err
		}
	}
	return nil
}

func aiConfigVariationAttributes(projectKey, configKey, variationKey string, tools []ldapi.VariationTool) map[string]string {
	attributes := map[string]string{
		"project_key": projectKey,
		"config_key":  configKey,
		"key":         variationKey,
	}
	if len(tools) > 0 {
		attributes["tool_keys.#"] = strconv.Itoa(len(tools))
		for _, tool := range tools {
			toolKey := tool.GetKey()
			attributes[fmt.Sprintf("tool_keys.%d", terraformutils.HashString(toolKey))] = toolKey
		}
	}
	return attributes
}

func closeResponseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}
