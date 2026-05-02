// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

type EnvironmentGenerator struct {
	LaunchDarklyService
}

func (g *EnvironmentGenerator) loadEnvironments(ctx context.Context, client *ldapi.APIClient, projectKey string) error {
	envs, err := getEnvironments(ctx, client, projectKey)
	if err != nil {
		return err
	}
	for _, env := range envs {
		resource := terraformutils.NewResource(
			projectKey+"/"+env.Key,
			environmentResourceName(projectKey, env.Name, env.Key),
			"launchdarkly_environment",
			"launchdarkly",
			map[string]string{
				"key":         env.Key,
				"project_key": projectKey,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func environmentResourceName(projectKey, name, key string) string {
	return launchDarklyProjectResourceName(projectKey, name, key)
}

func (g *EnvironmentGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadEnvironments(ctx, client, project.Key); err != nil {
			return err
		}
	}
	return nil
}
