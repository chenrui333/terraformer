// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type SegmentGenerator struct {
	LaunchDarklyService
}

func (g *SegmentGenerator) loadSegment(ctx context.Context, client *ldapi.APIClient, project, envKey string) error {
	segments, _, err := client.SegmentsApi.GetSegments(ctx, project, envKey).Execute()
	if err != nil {
		return err
	}
	for _, segment := range segments.Items {
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
	projects, err := getProjects(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient))
	if err != nil {
		return err
	}
	for _, project := range projects.Items {
		if project.Environments == nil {
			continue
		}
		for _, env := range project.Environments.Items {
			if err := g.loadSegment(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient), project.Key, env.Key); err != nil {
				return err
			}
		}
	}

	return nil
}
