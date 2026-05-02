// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"
	"net/url"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type UserSchemaPropertyGenerator struct {
	OktaService
}

func (g UserSchemaPropertyGenerator) createResources(userSchema *okta.UserSchema, userTypeID string, userTypeName string) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for index := range userSchema.Definitions.Custom.Properties {
		resources = append(resources, terraformutils.NewResource(
			index,
			normalizeResourceName(userTypeName)+"_property_"+normalizeResourceName(index),
			"okta_user_schema_property",
			"okta",
			map[string]string{
				"index":     index,
				"user_type": userTypeID,
			},
			[]string{},
			map[string]interface{}{},
		))
	}

	for index := range userSchema.Definitions.Base.Properties {
		resources = append(resources, terraformutils.NewResource(
			index,
			normalizeResourceName(userTypeName)+"_property_"+normalizeResourceName(index),
			"okta_user_base_schema_property",
			"okta",
			map[string]string{
				"index":     index,
				"user_type": userTypeID,
			},
			[]string{},
			map[string]interface{}{},
		))
	}
	return resources
}

func (g *UserSchemaPropertyGenerator) InitResources() error {
	var resources []terraformutils.Resource
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	userTypes, err := getUserTypes(ctx, client)
	if err != nil {
		return err
	}

	for _, userType := range userTypes {
		schemaID := getUserTypeSchemaID(userType)
		if schemaID != "" {
			schema, _, err := client.UserSchema.GetUserSchema(ctx, schemaID)
			if err != nil {
				return err
			}

			userTypeID := "default"
			if userType.Name != "user" {
				userTypeID = userType.Id
			}

			resources = append(resources, g.createResources(schema, userTypeID, userType.Name)...)
		}
	}

	g.Resources = resources
	return nil
}

func getUserTypes(ctx context.Context, client *okta.Client) ([]*okta.UserType, error) {
	output, resp, err := client.UserType.ListUserTypes(ctx)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextUserTypeSet []*okta.UserType
		resp, err = resp.Next(ctx, &nextUserTypeSet)
		if err != nil {
			return nil, err
		}
		output = append(output, nextUserTypeSet...)
	}

	return output, nil
}

func getUserTypeSchemaID(ut *okta.UserType) string {
	fm, ok := ut.Links.(map[string]interface{})
	if ok {
		sm, ok := fm["schema"].(map[string]interface{})
		if ok {
			href, ok := sm["href"].(string)
			if ok {
				u, _ := url.Parse(href)
				return strings.TrimPrefix(u.EscapedPath(), "/api/v1/meta/schemas/user/")
			}
		}
	}
	return ""
}
