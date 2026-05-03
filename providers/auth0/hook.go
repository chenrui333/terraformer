// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	HookAllowEmptyValues = []string{}
)

type HookGenerator struct {
	Auth0Service
}

func (g HookGenerator) createResources(hooks []*management.Hook) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, hook := range hooks {
		if hook == nil {
			return nil, auth0MissingResource("auth0_hook")
		}
		resourceName, err := auth0RequiredString("auth0_hook", "id", hook.ID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(hook.Name, resourceName),
			"auth0_hook",
			"auth0",
			HookAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *HookGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	list := []*management.Hook{}

	var page int
	for {
		l, err := m.Hook.List(ctx, management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.Hooks...)
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
