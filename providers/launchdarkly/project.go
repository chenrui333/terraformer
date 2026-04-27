// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	launchdarkly "github.com/launchdarkly/api-client-go"
)

type ProjectGenerator struct {
	LaunchDarklyService
}

func getProjects(ctx context.Context, client *launchdarkly.APIClient) (launchdarkly.Projects, error) {
	projects, _, err := client.ProjectsApi.GetProjects(ctx)
	return projects, err
}

func (g *ProjectGenerator) loadProjects(ctx context.Context, client *launchdarkly.APIClient) error {
	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects.Items {
		resource := terraformutils.NewResource(
			project.Key,
			project.Key,
			"launchdarkly_project",
			"launchdarkly",
			map[string]string{
				"key": project.Key,
			},
			[]string{},
			map[string]interface{}{})
		resource.IgnoreKeys = append(resource.IgnoreKeys, "include_in_snippet")
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ProjectGenerator) InitResources() error {
	if err := g.loadProjects(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*launchdarkly.APIClient)); err != nil {
		return err
	}

	return nil
}
