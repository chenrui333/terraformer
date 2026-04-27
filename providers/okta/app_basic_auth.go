// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type AppBasicAuthGenerator struct {
	OktaService
}

func (g AppBasicAuthGenerator) createResources(appList []*okta.Application) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, app := range appList {
		resources = append(resources, terraformutils.NewSimpleResource(
			app.Id,
			normalizeResourceName(app.Id+"_"+app.Name),
			"okta_app_basic_auth",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *AppBasicAuthGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	apps, err := getBasicAuthApplications(ctx, client)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(apps)
	return nil
}

func getBasicAuthApplications(ctx context.Context, client *okta.Client) ([]*okta.Application, error) {
	signOnMode := "BASIC_AUTH"
	apps, err := getApplications(ctx, client, signOnMode)
	if err != nil {
		return nil, err
	}

	return apps, nil
}
