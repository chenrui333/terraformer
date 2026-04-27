// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

var (
	// UserAllowEmptyValues ...
	UserAllowEmptyValues = []string{}
)

// UserGenerator ...
type UserGenerator struct {
	DatadogService
}

func (g *UserGenerator) createResources(users []datadogV2.User) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, user := range users {
		relations := user.GetRelationships()
		roles := relations.GetRoles()
		// If no roles are present, we can assume user was created via the V1 API
		// Hence, import the user via their handle
		if len(roles.GetData()) == 0 {
			attr := user.GetAttributes()
			resources = append(resources, g.createResource(attr.GetHandle()))
			continue
		}

		resources = append(resources, g.createResource(user.GetId()))
	}
	return resources
}

func (g *UserGenerator) createResource(userID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		userID,
		fmt.Sprintf("user_%s", userID),
		"datadog_user",
		"datadog",
		UserAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each user create 1 TerraformResource.
// Need User ID as ID for terraform resource
func (g *UserGenerator) InitResources() error {
	var users []datadogV2.User
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewUsersApi(datadogClient)

	pageSize := int64(1000)
	pageNumber := int64(0)
	remaining := int64(1)
	optionalParams := datadogV2.NewListUsersOptionalParameters()
	for _, filter := range g.Filter {
		if filter.IsApplicable("user") && filter.FieldPath == "disabled" {
			if len(filter.AcceptableValues) == 1 && strings.ToLower(filter.AcceptableValues[0]) == "false" {
				optionalParams = optionalParams.WithFilterStatus("Active,Pending")
			}
		}
	}

	for remaining > int64(0) {
		resp, _, err := api.ListUsers(auth, *optionalParams.
			WithPageSize(pageSize).
			WithPageNumber(pageNumber))
		if err != nil {
			return err
		}
		users = append(users, resp.GetData()...)

		remaining = resp.Meta.Page.GetTotalCount() - pageSize*(pageNumber+1)
		pageNumber++
	}

	g.Resources = g.createResources(users)
	return nil
}
