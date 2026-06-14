// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"
	"fmt"
	"sort"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	TriggerBindingAllowEmptyValues = []string{}
)

type TriggerBindingGenerator struct {
	Auth0Service
}

func (g TriggerBindingGenerator) createResources(bindings map[string][]*management.ActionBinding) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	triggerIDs := make([]string, 0, len(bindings))
	for triggerID := range bindings {
		triggerIDs = append(triggerIDs, triggerID)
	}
	sort.Strings(triggerIDs)

	for _, triggerID := range triggerIDs {
		actions, err := auth0TriggerActions(bindings[triggerID])
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewResource(
			triggerID,
			triggerID,
			"auth0_trigger_actions",
			"auth0",
			map[string]string{},
			TriggerBindingAllowEmptyValues,
			map[string]interface{}{
				"trigger": triggerID,
				"actions": actions,
			},
		))
	}
	return resources, nil
}

func auth0TriggerActions(bindings []*management.ActionBinding) ([]map[string]interface{}, error) {
	actions := make([]map[string]interface{}, 0, len(bindings))
	for _, binding := range bindings {
		if binding == nil {
			return nil, auth0MissingResource("auth0_trigger_actions binding")
		}
		actionID := ""
		if binding.Action != nil {
			actionID = binding.Action.GetID()
		}
		if actionID == "" {
			return nil, fmt.Errorf("%s resource is missing %s", "auth0_trigger_actions", "action_id")
		}
		displayName := binding.GetDisplayName()
		if displayName == "" {
			displayName = actionID
		}
		actions = append(actions, map[string]interface{}{
			"id":           actionID,
			"display_name": displayName,
		})
	}
	return actions, nil
}

func (g *TriggerBindingGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	bindings := map[string][]*management.ActionBinding{}

	t, err := m.Actions.Triggers.List(ctx)
	if err != nil {
		return err
	}

	for _, trigger := range t.Triggers {
		if trigger == nil {
			return auth0MissingResource("auth0_trigger_actions trigger")
		}
		triggerID := trigger.GetID()
		if triggerID == "" {
			return fmt.Errorf("%s resource is missing %s", "auth0_trigger_actions", "trigger_id")
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
				return auth0MissingResource("auth0_trigger_actions binding")
			}
			bindings[string(triggerID)] = append(bindings[string(triggerID)], binding)
		}
	}

	resources, err := g.createResources(bindings)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
