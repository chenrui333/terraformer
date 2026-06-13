// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

const oktaUserSchemaPathPrefix = "/api/v1/meta/schemas/user/"

type UserSchemaPropertyGenerator struct {
	OktaService
}

func (g UserSchemaPropertyGenerator) createResources(userSchema *okta.UserSchema, userTypeID string, userTypeName string) []terraformutils.Resource {
	var resources []terraformutils.Resource
	definitions := userSchema.GetDefinitions()
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

	var basePropertyNames []string
	if base, ok := definitions.GetBaseOk(); ok {
		basePropertyNames = userSchemaBasePropertyNames(base.GetProperties())
	}
	for _, index := range basePropertyNames {
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
		schemaID, err := getUserTypeSchemaID(userType)
		if err != nil {
			return err
		}
		if schemaID != "" {
			schema, _, err := client.SchemaAPI.GetUserSchema(ctx, schemaID).Execute()
			if err != nil {
				return err
			}

			userTypeName := getUserTypeName(userType)
			userTypeID := getUserTypeResourceID(userType)

			resources = append(resources, g.createResources(schema, userTypeID, userTypeName)...)
		}
	}

	g.Resources = resources
	return nil
}

func getUserTypes(ctx context.Context, client *okta.APIClient) ([]okta.UserType, error) {
	output, resp, err := client.UserTypeAPI.ListUserTypes(ctx).Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextUserTypeSet []okta.UserType
		resp, err = resp.Next(&nextUserTypeSet)
		if err != nil {
			return nil, err
		}
		output = append(output, nextUserTypeSet...)
	}

	return output, nil
}

func getUserTypeName(ut okta.UserType) string {
	if name := getUserTypeAPIName(ut); name != "" {
		return name
	}
	if displayName, ok := ut.AdditionalProperties["displayName"].(string); ok && displayName != "" {
		return displayName
	}
	return ut.GetId()
}

func getUserTypeAPIName(ut okta.UserType) string {
	if name, ok := ut.AdditionalProperties["name"].(string); ok {
		return name
	}
	return ""
}

func getUserTypeResourceID(ut okta.UserType) string {
	if getUserTypeAPIName(ut) == "user" {
		return "default"
	}
	return ut.GetId()
}

func getUserTypeSchemaID(ut okta.UserType) (string, error) {
	links, ok := ut.AdditionalProperties["_links"]
	if !ok {
		return "", nil
	}
	fm, ok := links.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("parse Okta user type %q schema link: links has type %T, want map[string]interface{}", ut.GetId(), links)
	}
	schemaValue, ok := fm["schema"]
	if !ok {
		return "", nil
	}
	sm, ok := schemaValue.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("parse Okta user type %q schema link: schema has type %T, want map[string]interface{}", ut.GetId(), schemaValue)
	}
	hrefValue, ok := sm["href"]
	if !ok {
		return "", nil
	}
	href, ok := hrefValue.(string)
	if !ok {
		return "", fmt.Errorf("parse Okta user type %q schema link: href has type %T, want string", ut.GetId(), hrefValue)
	}
	u, err := url.Parse(href)
	if err != nil {
		return "", fmt.Errorf("parse Okta user type %q schema link: %w", ut.GetId(), err)
	}
	path := u.EscapedPath()
	if !strings.HasPrefix(path, oktaUserSchemaPathPrefix) {
		return "", fmt.Errorf("parse Okta user type %q schema link %q: unexpected path %q", ut.GetId(), href, path)
	}
	schemaID := strings.TrimPrefix(path, oktaUserSchemaPathPrefix)
	if schemaID == "" {
		return "", fmt.Errorf("parse Okta user type %q schema link %q: missing schema ID", ut.GetId(), href)
	}
	return schemaID, nil
}

func userSchemaBasePropertyNames(properties okta.UserSchemaBaseProperties) []string {
	names := make([]string, 0, len(properties.AdditionalProperties))
	value := reflect.ValueOf(properties)
	typeOfProperties := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typeOfProperties.Field(i)
		if field.Name == "AdditionalProperties" {
			continue
		}
		jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonName == "" || jsonName == "-" {
			continue
		}
		if !value.Field(i).IsNil() {
			names = append(names, jsonName)
		}
	}
	for index := range properties.AdditionalProperties {
		names = append(names, index)
	}
	sort.Strings(names)
	return names
}
