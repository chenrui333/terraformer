// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type FlagTemplatesGenerator struct {
	LaunchDarklyService
}

func (g *FlagTemplatesGenerator) loadFlagTemplates(ctx context.Context, client *ldapi.APIClient, projectKey string) error {
	if _, _, err := client.ProjectsApi.GetFlagDefaultsByProject(ctx, projectKey).Execute(); err != nil {
		return err
	}
	resource := terraformutils.NewResource(
		projectKey,
		projectKey,
		"launchdarkly_flag_templates",
		"launchdarkly",
		map[string]string{
			"project_key": projectKey,
		},
		[]string{},
		map[string]interface{}{})
	g.Resources = append(g.Resources, resource)
	return nil
}

func (g *FlagTemplatesGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadFlagTemplates(ctx, client, project.Key); err != nil {
			return err
		}
	}
	return nil
}
