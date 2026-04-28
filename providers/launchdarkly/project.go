// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

const pageSize = 20

type ProjectGenerator struct {
	LaunchDarklyService
}

func getProjects(ctx context.Context, client *ldapi.APIClient) ([]ldapi.Project, error) {
	var allProjects []ldapi.Project
	for offset := int64(0); ; offset += pageSize {
		projects, _, err := client.ProjectsApi.GetProjects(ctx).
			Limit(pageSize).
			Offset(offset).
			Execute()
		if err != nil {
			return nil, err
		}
		allProjects = append(allProjects, projects.Items...)
		if projects.TotalCount == nil || int64(len(allProjects)) >= int64(*projects.TotalCount) {
			break
		}
	}
	return allProjects, nil
}

func (g *ProjectGenerator) loadProjects(ctx context.Context, client *ldapi.APIClient) error {
	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
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
	if err := g.loadProjects(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient)); err != nil {
		return err
	}

	return nil
}
