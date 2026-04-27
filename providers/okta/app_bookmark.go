// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v5/okta"
)

type AppBookmarkGenerator struct {
	OktaService
}

func (g *AppBookmarkGenerator) createResources(appList []okta.ListApplications200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, app := range appList {
		if app.BookmarkApplication != nil {
			if id, label := app.BookmarkApplication.Id, app.BookmarkApplication.Label; id != nil && label != "" {
				resources = append(resources, terraformutils.NewSimpleResource(
					*id,
					normalizeResourceName(*id+"_"+label),
					"okta_app_bookmark",
					"okta",
					[]string{},
				))
			}
		}
	}
	return resources
}

func (g *AppBookmarkGenerator) InitResources() error {
	ctx, client, err := g.ClientV5()
	if err != nil {
		return err
	}

	appList, resp, err := client.ApplicationAPI.ListApplications(ctx).Execute()
	if err != nil {
		return fmt.Errorf("error listing applications: %w", err)
	}

	allApplications := appList

	for resp.HasNextPage() {
		var nextAppList []okta.ListApplications200ResponseInner
		resp, err = resp.Next(&nextAppList)
		if err != nil {
			return fmt.Errorf("error fetching next page: %w", err)
		}
		allApplications = append(allApplications, nextAppList...)
	}

	g.Resources = g.createResources(allApplications)
	return nil
}
