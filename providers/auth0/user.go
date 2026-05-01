// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	UserAllowEmptyValues = []string{}
)

type UserGenerator struct {
	Auth0Service
}

func (g UserGenerator) createResources(users []*management.User) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, user := range users {
		resourceName := user.ID
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
	m := g.generateClient()
	ctx := context.Background()
	list := []*management.User{}

	var page int
	for {
		l, err := m.User.List(ctx, management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.Users...)
		if !l.HasNext() {
			break
		}
		page++
	}

	g.Resources = g.createResources(list)
	return nil
}
