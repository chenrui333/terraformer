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

func (g UserGenerator) createResources(users []*management.UserResponseSchema) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, user := range users {
		resourceName := user.UserID
		resources = append(resources, terraformutils.NewSimpleResource(
			*resourceName,
			*resourceName,
			"auth0_user",
			"auth0",
			UserAllowEmptyValues,
		))
	}
	return resources
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

	g.Resources = g.createResources(list)
	return nil
}
