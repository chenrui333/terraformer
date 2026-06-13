// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"sort"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type AppUserSchemaPropertyGenerator struct {
	OktaService
}

func (g AppUserSchemaPropertyGenerator) createResources(appUserSchema *okta.UserSchema, appID string) []terraformutils.Resource {
	var resources []terraformutils.Resource
	definitions := appUserSchema.GetDefinitions()
	var customPropertyNames []string
	if custom, ok := definitions.GetCustomOk(); ok {
		for index := range custom.GetProperties() {
			customPropertyNames = append(customPropertyNames, index)
		}
	}
	sort.Strings(customPropertyNames)
	for _, index := range customPropertyNames {
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

	var basePropertyNames []string
	if base, ok := definitions.GetBaseOk(); ok {
		basePropertyNames = userSchemaBasePropertyNames(base.GetProperties())
	}
	for _, index := range basePropertyNames {
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
		appSummary, _ := getApplicationSummary(app)
		appUserSchema, _, err := client.SchemaAPI.GetApplicationUserSchema(ctx, appSummary.ID).Execute()
		if err != nil {
			return err
		}

		resources = append(resources, g.createResources(appUserSchema, appSummary.ID)...)
	}
	g.Resources = resources
	return nil
}
