// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

const launchDarklyViewAPIVersion = "beta"

type ViewGenerator struct {
	LaunchDarklyService
}

type ViewLinksGenerator struct {
	LaunchDarklyService
}

func getViews(ctx context.Context, client *ldapi.APIClient, projectKey string) ([]ldapi.View, error) {
	var allViews []ldapi.View
	for offset := int32(0); ; offset += pageSize {
		views, resp, err := client.ViewsBetaApi.GetViews(ctx, projectKey).
			LDAPIVersion(launchDarklyViewAPIVersion).
			Limit(pageSize).
			Offset(offset).
			Execute()
		closeResponseBody(resp)
		if err != nil {
			return nil, err
		}
		if views == nil {
			break
		}
		items := views.GetItems()
		allViews = append(allViews, items...)
		if len(items) == 0 || int64(len(allViews)) >= int64(views.GetTotalCount()) {
			break
		}
	}
	return allViews, nil
}

func getViewLinkedResources(ctx context.Context, client *ldapi.APIClient, projectKey, viewKey, resourceType string) ([]ldapi.ViewLinkedResource, error) {
	var allResources []ldapi.ViewLinkedResource
	for offset := int32(0); ; offset += pageSize {
		resources, resp, err := client.ViewsBetaApi.GetLinkedResources(ctx, projectKey, viewKey, resourceType).
			LDAPIVersion(launchDarklyViewAPIVersion).
			Limit(pageSize).
			Offset(offset).
			Execute()
		closeResponseBody(resp)
		if err != nil {
			return nil, err
		}
		if resources == nil {
			break
		}
		items := resources.GetItems()
		allResources = append(allResources, items...)
		if len(items) == 0 || int64(len(allResources)) >= int64(resources.GetTotalCount()) {
			break
		}
	}
	return allResources, nil
}

func (g *ViewGenerator) loadViews(ctx context.Context, client *ldapi.APIClient, projectKey string) error {
	views, err := getViews(ctx, client, projectKey)
	if err != nil {
		return err
	}
	for _, view := range views {
		viewKey := view.GetKey()
		resource := terraformutils.NewResource(
			fmt.Sprintf("%s/%s", projectKey, viewKey),
			launchDarklyProjectResourceName(projectKey, view.GetName(), viewKey),
			"launchdarkly_view",
			"launchdarkly",
			map[string]string{
				"project_key": projectKey,
				"key":         viewKey,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ViewGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadViews(ctx, client, project.Key); err != nil {
			return err
		}
	}
	return nil
}

func (g *ViewLinksGenerator) loadViewLinks(ctx context.Context, client *ldapi.APIClient, projectKey string) error {
	views, err := getViews(ctx, client, projectKey)
	if err != nil {
		return err
	}
	for _, view := range views {
		viewKey := view.GetKey()
		linkedFlags, err := getViewLinkedResources(ctx, client, projectKey, viewKey, "flags")
		if err != nil {
			return err
		}
		linkedSegments, err := getViewLinkedResources(ctx, client, projectKey, viewKey, "segments")
		if err != nil {
			return err
		}
		if len(linkedFlags) == 0 && len(linkedSegments) == 0 {
			continue
		}
		resource := terraformutils.NewResource(
			fmt.Sprintf("%s/%s", projectKey, viewKey),
			fmt.Sprintf("%s-%s-links", projectKey, viewKey),
			"launchdarkly_view_links",
			"launchdarkly",
			map[string]string{
				"project_key": projectKey,
				"view_key":    viewKey,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ViewLinksGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	projects, err := getProjects(ctx, client)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := g.loadViewLinks(ctx, client, project.Key); err != nil {
			return err
		}
	}
	return nil
}
