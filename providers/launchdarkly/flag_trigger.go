// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type FlagTriggerGenerator struct {
	LaunchDarklyService
}

func getFeatureFlags(ctx context.Context, client *ldapi.APIClient, projectKey string) ([]ldapi.FeatureFlag, error) {
	var allFlags []ldapi.FeatureFlag
	for offset := int64(0); ; offset += pageSize {
		featureFlags, resp, err := client.FeatureFlagsApi.GetFeatureFlags(ctx, projectKey).
			Limit(pageSize).
			Offset(offset).
			Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			return nil, err
		}
		if featureFlags == nil {
			break
		}
		allFlags = append(allFlags, featureFlags.GetItems()...)
		if featureFlags.TotalCount == nil || int64(len(allFlags)) >= int64(*featureFlags.TotalCount) {
			break
		}
	}
	return allFlags, nil
}

func (g *FlagTriggerGenerator) loadFlagTriggers(ctx context.Context, client *ldapi.APIClient, projectKey, envKey, flagKey string) error {
	triggers, resp, err := client.FlagTriggersApi.GetTriggerWorkflows(ctx, projectKey, envKey, flagKey).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	if triggers == nil {
		return nil
	}
	for _, trigger := range triggers.GetItems() {
		triggerID := trigger.GetId()
		resource := terraformutils.NewResource(
			strings.Join([]string{projectKey, envKey, flagKey, triggerID}, "/"),
			fmt.Sprintf("%s-%s-%s-%s", projectKey, envKey, flagKey, triggerID),
			"launchdarkly_flag_trigger",
			"launchdarkly",
			map[string]string{
				"project_key":     projectKey,
				"env_key":         envKey,
				"flag_key":        flagKey,
				"integration_key": trigger.GetIntegrationKey(),
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *FlagTriggerGenerator) InitResources() error {
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
		flags, err := getFeatureFlags(ctx, client, project.Key)
		if err != nil {
			return err
		}
		for _, env := range envs {
			for _, flag := range flags {
				if err := g.loadFlagTriggers(ctx, client, project.Key, env.Key, flag.Key); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
