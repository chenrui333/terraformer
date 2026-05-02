// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

type SegmentGenerator struct {
	LaunchDarklyService
}

func getEnvironments(ctx context.Context, client *ldapi.APIClient, projectKey string) ([]ldapi.Environment, error) {
	var allEnvs []ldapi.Environment
	for offset := int64(0); ; offset += pageSize {
		envs, _, err := client.EnvironmentsApi.GetEnvironmentsByProject(ctx, projectKey).
			Limit(pageSize).
			Offset(offset).
			Execute()
		if err != nil {
			return nil, err
		}
		allEnvs = append(allEnvs, envs.Items...)
		if envs.TotalCount == nil || int64(len(allEnvs)) >= int64(*envs.TotalCount) {
			break
		}
	}
	return allEnvs, nil
}

func (g *SegmentGenerator) loadSegment(ctx context.Context, client *ldapi.APIClient, project, envKey string) error {
	var allSegments []ldapi.UserSegment
	for offset := int64(0); ; offset += pageSize {
		segments, _, err := client.SegmentsApi.GetSegments(ctx, project, envKey).
			Limit(pageSize).
			Offset(offset).
			Execute()
		if err != nil {
			return err
		}
		allSegments = append(allSegments, segments.Items...)
		if segments.TotalCount == nil || int64(len(allSegments)) >= int64(*segments.TotalCount) {
			break
		}
	}
	for _, segment := range allSegments {
		resource := terraformutils.NewResource(
			segment.Key,
			project+"-"+envKey+"-"+segment.Name,
			"launchdarkly_segment",
			"launchdarkly",
			map[string]string{
				"key":         segment.Key,
				"project_key": project,
				"env_key":     envKey,
			},
			[]string{},
			map[string]interface{}{})
		resource.IgnoreKeys = append(resource.IgnoreKeys, "include_in_snippet")
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *SegmentGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		envs, err := getEnvironments(ctx, client, project.Key)
		if err != nil {
			return err
		}
		for _, env := range envs {
			if err := g.loadSegment(ctx, client, project.Key, env.Key); err != nil {
				return err
			}
		}
	}

	return nil
}
