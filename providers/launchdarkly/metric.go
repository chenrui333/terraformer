// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"fmt"
	"net/url"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

type MetricGenerator struct {
	LaunchDarklyService
}

func (g *MetricGenerator) loadMetrics(ctx context.Context, apiKey, projectKey string) error {
	path := fmt.Sprintf("/metrics/%s?limit=%d", url.PathEscape(projectKey), launchDarklyDefaultPageSize)
	for path != "" {
		metrics := &ldapi.MetricCollectionRep{}
		if err := getLaunchDarklyAPI(ctx, apiKey, path, metrics); err != nil {
			return err
		}

		for _, metric := range metrics.GetItems() {
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
		path = nextPagePath(metrics.GetLinks())
	}
	return nil
}

func (g *MetricGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	apiKey := g.GetArgs()["api_key"].(string)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadMetrics(ctx, apiKey, project.Key); err != nil {
			return err
		}
	}
	return nil
}
