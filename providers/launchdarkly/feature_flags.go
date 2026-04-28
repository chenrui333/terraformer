// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

var featureFlagsAllowEmptyValues = []string{"variations.*.value"}

type FeatureFlagsGenerator struct {
	LaunchDarklyService
}

func (g *FeatureFlagsGenerator) loadFeatureFlagEnv(ctx context.Context, client *ldapi.APIClient, projectKey, flagKey string) error {
	ff, _, err := client.FeatureFlagsApi.GetFeatureFlag(ctx, projectKey, flagKey).Execute()
	if err != nil {
		return err
	}
	for envKey := range ff.Environments {
		resource := terraformutils.NewResource(
			projectKey+"/"+envKey+"/"+flagKey,
			projectKey+"-"+envKey+"-"+flagKey,
			"launchdarkly_feature_flag_environment",
			"launchdarkly",
			map[string]string{
				"env_key": envKey,
				"flag_id": projectKey + "/" + flagKey,
			},
			featureFlagsAllowEmptyValues,
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *FeatureFlagsGenerator) loadFeatureFlags(ctx context.Context, client *ldapi.APIClient, project string) error {
	featureFlags, _, err := client.FeatureFlagsApi.GetFeatureFlags(ctx, project).Execute()
	if err != nil {
		return err
	}
	for _, featureFlag := range featureFlags.Items {
		resource := terraformutils.NewResource(
			featureFlag.Key,
			project+"-"+featureFlag.Name,
			"launchdarkly_feature_flag",
			"launchdarkly",
			map[string]string{
				"key":         featureFlag.Key,
				"project_key": project,
			},
			featureFlagsAllowEmptyValues,
			map[string]interface{}{})
		resource.IgnoreKeys = append(resource.IgnoreKeys, "include_in_snippet")
		err = g.loadFeatureFlagEnv(ctx, client, project, featureFlag.Key)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *FeatureFlagsGenerator) InitResources() error {
	projects, err := getProjects(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient))
	if err != nil {
		return err
	}
	for _, project := range projects.Items {
		if err := g.loadFeatureFlags(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient), project.Key); err != nil {
			return err
		}
	}
	return nil
}
