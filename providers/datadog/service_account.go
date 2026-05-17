// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// ServiceAccountAllowEmptyValues ...
	ServiceAccountAllowEmptyValues = []string{}
)

// ServiceAccountGenerator ...
type ServiceAccountGenerator struct {
	DatadogService
}

func (g *ServiceAccountGenerator) createResource(accountID string) (terraformutils.Resource, error) {
	if accountID == "" {
		return terraformutils.Resource{}, fmt.Errorf("service account missing id")
	}

	return terraformutils.NewSimpleResource(
		accountID,
		fmt.Sprintf("service_account_%s", accountID),
		"datadog_service_account",
		"datadog",
		ServiceAccountAllowEmptyValues,
	), nil
}

func (g *ServiceAccountGenerator) createResources(users []datadogV2.User) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, user := range users {
		attrs := user.GetAttributes()
		if !attrs.GetServiceAccount() {
			continue
		}
		resource, err := g.createResource(user.GetId())
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each service_account create 1 TerraformResource.
func (g *ServiceAccountGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewUsersApi(datadogClient)

	// Handle ID filter
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("service_account") {
			var resources []terraformutils.Resource
			for _, value := range filter.AcceptableValues {
				resource, err := g.createResource(value)
				if err != nil {
					return err
				}
				resources = append(resources, resource)
			}
			g.Resources = resources
			return nil
		}
	}

	// List all service accounts with pagination
	users, err := g.listServiceAccounts(auth, api)
	if err != nil {
		return err
	}

	resources, err := g.createResources(users)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *ServiceAccountGenerator) listServiceAccounts(auth context.Context, api *datadogV2.UsersApi) ([]datadogV2.User, error) {
	var users []datadogV2.User
	pageSize := int64(100)
	pageNumber := int64(0)
	remaining := int64(1)

	for remaining > int64(0) {
		optionalParams := datadogV2.NewListUsersOptionalParameters().
			WithPageSize(pageSize).
			WithPageNumber(pageNumber)

		resp, httpResp, err := api.ListUsers(auth, *optionalParams)
		if httpResp != nil && httpResp.Body != nil {
			httpResp.Body.Close()
		}
		if err != nil {
			return nil, err
		}
		users = append(users, resp.GetData()...)

		remaining = resp.Meta.Page.GetTotalCount() - pageSize*(pageNumber+1)
		pageNumber++
	}

	return users, nil
}
