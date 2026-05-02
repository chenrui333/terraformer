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

func (g ActionGenerator) createResources(actions []*management.Action) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, action := range actions {
		resourceName := *action.ID
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName+"_"+*action.Name,
			"auth0_action",
			"auth0",
			ActionAllowEmptyValues,
		))
	}
	return resources
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

	g.Resources = g.createResources(list)
	return nil
}
