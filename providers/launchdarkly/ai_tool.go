// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

type AIToolGenerator struct {
	LaunchDarklyService
}

func (g *AIToolGenerator) loadAITools(ctx context.Context, client *ldapi.APIClient, projectKey string) error {
	var allTools []ldapi.AITool
	for offset := int32(0); ; offset += pageSize {
		tools, resp, err := client.AIConfigsApi.ListAITools(ctx, projectKey).
			Limit(pageSize).
			Offset(offset).
			Execute()
		closeResponseBody(resp)
		if err != nil {
			return err
		}
		if tools == nil {
			break
		}
		allTools = append(allTools, tools.GetItems()...)
		if int64(len(allTools)) >= int64(tools.GetTotalCount()) {
			break
		}
	}
	for _, tool := range allTools {
		toolKey := tool.GetKey()
		resource := terraformutils.NewResource(
			fmt.Sprintf("%s/%s", projectKey, toolKey),
			fmt.Sprintf("%s-%s", projectKey, toolKey),
			"launchdarkly_ai_tool",
			"launchdarkly",
			map[string]string{
				"project_key": projectKey,
				"key":         toolKey,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *AIToolGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadAITools(ctx, client, project.Key); err != nil {
			return err
		}
	}
	return nil
}
