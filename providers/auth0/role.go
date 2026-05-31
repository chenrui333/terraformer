// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	RoleAllowEmptyValues = []string{}
)

type RoleGenerator struct {
	Auth0Service
}

func (g RoleGenerator) createResources(roles []*management.Role) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, role := range roles {
		if role == nil {
			return nil, auth0MissingResource("auth0_role")
		}
		resourceName, err := auth0RequiredString("auth0_role", "id", role.ID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(role.Name, resourceName),
			"auth0_role",
			"auth0",
			RoleAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *RoleGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	page, err := m.Roles.List(ctx, &management.ListRolesRequestParameters{})
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
