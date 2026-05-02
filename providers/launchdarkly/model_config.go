// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

type ModelConfigGenerator struct {
	LaunchDarklyService
}

func (g *ModelConfigGenerator) loadModelConfigs(ctx context.Context, client *ldapi.APIClient, projectKey string) error {
	modelConfigs, resp, err := client.AIConfigsApi.ListModelConfigs(ctx, projectKey).Execute()
	closeResponseBody(resp)
	if err != nil {
		return err
	}
	for _, modelConfig := range modelConfigs {
		modelConfigKey := modelConfig.GetKey()
		resource := terraformutils.NewResource(
			fmt.Sprintf("%s/%s", projectKey, modelConfigKey),
			launchDarklyProjectResourceName(projectKey, modelConfig.GetName(), modelConfigKey),
			"launchdarkly_model_config",
			"launchdarkly",
			map[string]string{
				"project_key": projectKey,
				"key":         modelConfigKey,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ModelConfigGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadModelConfigs(ctx, client, project.Key); err != nil {
			return err
		}
	}
	return nil
}
