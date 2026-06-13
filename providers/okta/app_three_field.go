// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type AppThreeFieldGenerator struct {
	OktaService
}

func (g AppThreeFieldGenerator) createResources(appList []okta.ListApplications200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, app := range appList {
		summary, ok := getApplicationSummary(app)
		if !ok {
			continue
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			summary.ID,
			normalizeResourceName(summary.ID+"_"+summary.Name),
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

func getThreeFieldApplications(ctx context.Context, client *okta.APIClient) ([]okta.ListApplications200ResponseInner, error) {
	signOnMode := "BROWSER_PLUGIN"
	apps, err := getApplications(ctx, client, signOnMode)
	if err != nil {
		return nil, err
	}

	var threeFieldApps []okta.ListApplications200ResponseInner
	for _, app := range apps {
		summary, ok := getApplicationSummary(app)
		if ok && summary.Name == "template_swa3field" {
			threeFieldApps = append(threeFieldApps, app)
		}
	}

	return threeFieldApps, nil
}
