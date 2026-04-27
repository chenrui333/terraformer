// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type AppThreeFieldGenerator struct {
	OktaService
}

func (g AppThreeFieldGenerator) createResources(appList []*okta.Application) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, app := range appList {
		resources = append(resources, terraformutils.NewSimpleResource(
			app.Id,
			normalizeResourceName(app.Id+"_"+app.Name),
			"okta_app_three_field",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *AppThreeFieldGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	apps, err := getThreeFieldApplications(ctx, client)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(apps)
	return nil
}

func getThreeFieldApplications(ctx context.Context, client *okta.Client) ([]*okta.Application, error) {
	signOnMode := "BROWSER_PLUGIN"
	apps, err := getApplications(ctx, client, signOnMode)
	if err != nil {
		return nil, err
	}

	threeFieldApps := []*okta.Application{}
	for _, app := range apps {
		if app.Name == "template_swa3field" {
			threeFieldApps = append(threeFieldApps, app)
		}
	}

	return threeFieldApps, nil
}
