// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v5/okta"
)

type AppOAuthGenerator struct {
	OktaService
}

func (g *AppOAuthGenerator) createResources(appList []okta.ListApplications200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, app := range appList {
		if app.OpenIdConnectApplication != nil {
			if id, label := app.OpenIdConnectApplication.Id, app.OpenIdConnectApplication.Label; id != nil && label != "" {
				resources = append(resources, terraformutils.NewSimpleResource(
					*id,
					normalizeResourceName(*id+"_"+label),
					"okta_app_oauth",
					"okta",
					[]string{},
				))
			}
		}
	}
	return resources
}

func (g *AppOAuthGenerator) InitResources() error {
	ctx, client, err := g.ClientV5()
	if err != nil {
		return err
	}

	appList, resp, err := client.ApplicationAPI.ListApplications(ctx).Filter("name eq \"oidc_client\"").Execute()
	if err != nil {
		return fmt.Errorf("error listing OAuth applications: %w", err)
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

func (g *AppOAuthGenerator) PostConvertHook() error {
	for i := range g.Resources {
		g.Resources[i].Item = escapeDollar(g.Resources[i].Item)
	}
	return nil
}
