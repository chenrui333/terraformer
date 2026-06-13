// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type AppAutoLoginGenerator struct {
	OktaService
}

func (g AppAutoLoginGenerator) createResources(appList []okta.ListApplications200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, app := range appList {
		summary, ok := getApplicationSummary(app)
		if !ok {
			continue
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			summary.ID,
			normalizeResourceName(summary.ID+"_"+summary.Name),
			"okta_app_auto_login",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *AppAutoLoginGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	apps, err := getAutoLoginApplications(ctx, client)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(apps)
	return nil
}

func getAutoLoginApplications(ctx context.Context, client *okta.APIClient) ([]okta.ListApplications200ResponseInner, error) {
	signOnMode := "AUTO_LOGIN"
	apps, err := getApplications(ctx, client, signOnMode)
	if err != nil {
		return nil, err
	}

	return apps, nil
}
