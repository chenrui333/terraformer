// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

var (
	HookAllowEmptyValues = []string{}
)

type HookGenerator struct {
	Auth0Service
}

func (g HookGenerator) createResources(hooks []*management.Hook) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, hook := range hooks {
		resourceName := *hook.ID
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName+"_"+*hook.Name,
			"auth0_hook",
			"auth0",
			HookAllowEmptyValues,
		))
	}
	return resources
}

func (g *HookGenerator) InitResources() error {
	m := g.generateClient()
	list := []*management.Hook{}

	var page int
	for {
		l, err := m.Hook.List(management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.Hooks...)
		if !l.HasNext() {
			break
		}
		page++
	}

	g.Resources = g.createResources(list)
	return nil
}
