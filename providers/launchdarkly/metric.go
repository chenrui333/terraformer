// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type MetricGenerator struct {
	LaunchDarklyService
}

func (g *MetricGenerator) loadMetrics(ctx context.Context, client *ldapi.APIClient, projectKey string) error {
	metrics, _, err := client.MetricsApi.GetMetrics(ctx, projectKey).Execute()
	if err != nil {
		return err
	}
	if metrics == nil {
		return nil
	}
	for _, metric := range metrics.Items {
		resource := terraformutils.NewResource(
			projectKey+"/"+metric.Key,
			projectKey+"-"+metric.Name,
			"launchdarkly_metric",
			"launchdarkly",
			map[string]string{
				"key":         metric.Key,
				"project_key": projectKey,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *MetricGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadMetrics(ctx, client, project.Key); err != nil {
			return err
		}
	}
	return nil
}
