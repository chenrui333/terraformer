// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type EventHookGenerator struct {
	OktaService
}

func (g EventHookGenerator) createResources(eventHookList []*okta.EventHook) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, eventHook := range eventHookList {
		resources = append(resources, terraformutils.NewSimpleResource(
			eventHook.Id,
			"event_hook_"+eventHook.Name,
			"okta_event_hook",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *EventHookGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, resp, err := client.EventHook.ListEventHooks(ctx)
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextEventHookSet []*okta.EventHook
		resp, err = resp.Next(ctx, &nextEventHookSet)
		if err != nil {
			return err
		}
		output = append(output, nextEventHookSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}
