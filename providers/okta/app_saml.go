// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v5/okta"
)

type AppSamlGenerator struct {
	OktaService
}

func (g *AppSamlGenerator) createResources(appList []okta.ListApplications200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, app := range appList {
		if app.SamlApplication != nil && app.SamlApplication.Id != nil && app.SamlApplication.Label != "" {
			resources = append(resources, terraformutils.NewSimpleResource(
				*app.SamlApplication.Id,
				normalizeResourceName(*app.SamlApplication.Id+"_"+app.SamlApplication.Label),
				"okta_app_saml",
				"okta",
				[]string{},
			))
		}

		if app.Saml11Application != nil && app.Saml11Application.Id != nil && app.Saml11Application.Label != "" {
			resources = append(resources, terraformutils.NewSimpleResource(
				*app.Saml11Application.Id,
				normalizeResourceName(*app.Saml11Application.Id+"_"+app.Saml11Application.Label),
				"okta_app_saml",
				"okta",
				[]string{},
			))
		}
	}
	return resources
}

func (g *AppSamlGenerator) InitResources() error {
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

func (g *AppSamlGenerator) PostConvertHook() error {
	for i := range g.Resources {
		g.Resources[i].Item = escapeDollar(g.Resources[i].Item)
	}
	return nil
}
