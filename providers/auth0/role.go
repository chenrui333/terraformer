// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

var (
	RoleAllowEmptyValues = []string{}
)

type RoleGenerator struct {
	Auth0Service
}

func (g RoleGenerator) createResources(roles []*management.Role) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, role := range roles {
		resourceName := *role.ID
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName+"_"+*role.Name,
			"auth0_role",
			"auth0",
			RoleAllowEmptyValues,
		))
	}
	return resources
}

func (g *RoleGenerator) InitResources() error {
	m := g.generateClient()
	list := []*management.Role{}

	var page int
	for {
		l, err := m.Role.List(management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.Roles...)
		if !l.HasNext() {
			break
		}
		page++
	}

	g.Resources = g.createResources(list)
	return nil
}
