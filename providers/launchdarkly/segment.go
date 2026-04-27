// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	launchdarkly "github.com/launchdarkly/api-client-go"
)

type SegmentGenerator struct {
	LaunchDarklyService
}

func (g *SegmentGenerator) loadSegment(ctx context.Context, client *launchdarkly.APIClient, project, envKey string) error {
	segments, _, err := client.UserSegmentsApi.GetUserSegments(ctx, project, envKey, &launchdarkly.UserSegmentsApiGetUserSegmentsOpts{})
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
	projects, err := getProjects(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*launchdarkly.APIClient))
	if err != nil {
		return err
	}
	for _, project := range projects.Items {
		for _, env := range project.Environments {
			if err := g.loadSegment(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*launchdarkly.APIClient), project.Key, env.Key); err != nil {
				return err
			}
		}
	}

	return nil
}
