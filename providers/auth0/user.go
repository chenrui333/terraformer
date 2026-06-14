// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	UserAllowEmptyValues = []string{}
)

type UserGenerator struct {
	Auth0Service
}

func (g UserGenerator) createResources(users []*management.UserResponseSchema) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, user := range users {
		if user == nil {
			return nil, auth0MissingResource("auth0_user")
		}
		resourceName, err := auth0RequiredString("auth0_user", "user_id", user.UserID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName,
			"auth0_user",
			"auth0",
			UserAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *UserGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	page, err := m.Users.List(ctx, &management.ListUsersRequestParameters{})
	if err != nil {
		return err
	}
	list, err := auth0PageResults(ctx, page)
	if err != nil {
		return err
	}

	resources, err := g.createResources(list)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
