// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	TriggerBindingAllowEmptyValues = []string{}
)

type TriggerBindingGenerator struct {
	Auth0Service
}

func (g TriggerBindingGenerator) createResources(bindings map[string]*management.ActionBinding) []terraformutils.Resource {
	resources := []terraformutils.Resource{}

	for _, binding := range bindings {
		resourceName := *binding.TriggerID
		resources = append(resources, terraformutils.NewResource(
			resourceName,
			*binding.ID,
			"auth0_trigger_binding",
			"auth0",
			map[string]string{},
			TriggerBindingAllowEmptyValues,
			map[string]interface{}{
				"trigger": *binding.TriggerID,
			},
		))
	}
	return resources
}

func (g *TriggerBindingGenerator) InitResources() error {
	m := g.generateClient()
	ctx := context.Background()
	bindings := map[string]*management.ActionBinding{}

	t, err := m.Action.Triggers(ctx)
	if err != nil {
		return err
	}

	for _, trigger := range t.Triggers {
		var page int
		for {
			l, err := m.Action.Bindings(ctx, *trigger.ID, management.Page(page))
			if err != nil {
				return err
			}
			for _, binding := range l.Bindings {
				if _, ok := bindings[*binding.ID]; !ok {
					bindings[*binding.ID] = binding
				}
			}
			if !l.HasNext() {
				break
			}
			page++
		}
	}

	g.Resources = g.createResources(bindings)
	return nil
}
