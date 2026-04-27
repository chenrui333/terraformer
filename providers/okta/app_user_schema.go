// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type AppUserSchemaPropertyGenerator struct {
	OktaService
}

func (g AppUserSchemaPropertyGenerator) createResources(appUserSchema *okta.UserSchema, appID string) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for index := range appUserSchema.Definitions.Custom.Properties {
		resources = append(resources, terraformutils.NewResource(
			index,
			normalizeResourceName(appID)+"_property_"+normalizeResourceName(index),
			"okta_app_user_schema_property",
			"okta",
			map[string]string{
				"app_id": appID,
				"index":  index,
			},
			[]string{},
			map[string]interface{}{},
		))
	}

	for index := range appUserSchema.Definitions.Base.Properties {
		resources = append(resources, terraformutils.NewResource(
			index,
			normalizeResourceName(appID)+"_property_"+normalizeResourceName(index),
			"okta_app_user_base_schema_property",
			"okta",
			map[string]string{
				"app_id": appID,
				"index":  index,
			},
			[]string{},
			map[string]interface{}{},
		))
	}
	return resources
}

func (g *AppUserSchemaPropertyGenerator) InitResources() error {
	var resources []terraformutils.Resource
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	apps, err := getAllApplications(ctx, client)
	if err != nil {
		return err
	}

	for _, app := range apps {
		appUserSchema, _, err := client.UserSchema.GetApplicationUserSchema(ctx, app.Id)
		if err != nil {
			return err
		}

		resources = append(resources, g.createResources(appUserSchema, app.Id)...)
	}
	g.Resources = resources
	return nil
}
