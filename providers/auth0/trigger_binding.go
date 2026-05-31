// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"
	"fmt"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	TriggerBindingAllowEmptyValues = []string{}
)

type TriggerBindingGenerator struct {
	Auth0Service
}

func (g TriggerBindingGenerator) createResources(bindings map[string]*management.ActionBinding) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}

	for _, binding := range bindings {
		if binding == nil {
			return nil, auth0MissingResource("auth0_trigger_binding")
		}
		triggerID := binding.GetTriggerID()
		if triggerID == "" {
			return nil, fmt.Errorf("%s resource is missing %s", "auth0_trigger_binding", "trigger_id")
		}
		resourceID := string(triggerID)
		resourceName, err := auth0RequiredString("auth0_trigger_binding", "id", binding.ID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewResource(
			resourceID,
			resourceName,
			"auth0_trigger_binding",
			"auth0",
			map[string]string{},
			TriggerBindingAllowEmptyValues,
			map[string]interface{}{
				"trigger": resourceID,
			},
		))
	}
	return resources, nil
}

func (g *TriggerBindingGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	bindings := map[string]*management.ActionBinding{}

	t, err := m.Actions.Triggers.List(ctx)
	if err != nil {
		return err
	}

	for _, trigger := range t.Triggers {
		if trigger == nil {
			return auth0MissingResource("auth0_trigger_binding trigger")
		}
		triggerID := trigger.GetID()
		if triggerID == "" {
			return fmt.Errorf("%s resource is missing %s", "auth0_trigger_binding", "trigger_id")
		}
		page, err := m.Actions.Triggers.Bindings.List(ctx, &triggerID, &management.ListActionTriggerBindingsRequestParameters{})
		if err != nil {
			return err
		}
		list, err := auth0PageResults(ctx, page)
		if err != nil {
			return err
		}
		for _, binding := range list {
			if binding == nil {
				return auth0MissingResource("auth0_trigger_binding")
			}
			bindingID, err := auth0RequiredString("auth0_trigger_binding", "id", binding.ID)
			if err != nil {
				return err
			}
			if _, ok := bindings[bindingID]; !ok {
				bindings[bindingID] = binding
			}
		}
	}

	resources, err := g.createResources(bindings)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
