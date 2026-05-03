// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	ActionAllowEmptyValues = []string{}
)

type ActionGenerator struct {
	Auth0Service
}

func (g ActionGenerator) createResources(actions []*management.Action) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, action := range actions {
		if action == nil {
			return nil, auth0MissingResource("auth0_action")
		}
		resourceName, err := auth0RequiredString("auth0_action", "id", action.ID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(action.Name, resourceName),
			"auth0_action",
			"auth0",
			ActionAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *ActionGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	list := []*management.Action{}

	var page int
	for {
		l, err := m.Action.List(ctx, management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.Actions...)
		if !l.HasNext() {
			break
		}
		page++
	}

	resources, err := g.createResources(list)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
