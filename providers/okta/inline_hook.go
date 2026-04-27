// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type InlineHookGenerator struct {
	OktaService
}

func (g InlineHookGenerator) createResources(inlineHookList []*okta.InlineHook) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, inlineHook := range inlineHookList {
		resources = append(resources, terraformutils.NewSimpleResource(
			inlineHook.Id,
			"inline_hook_"+inlineHook.Name,
			"okta_inline_hook",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *InlineHookGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, resp, err := client.InlineHook.ListInlineHooks(ctx, nil)
	if err != nil {
		return e
	}

	for resp.HasNextPage() {
		var nextInlineHookSet []*okta.InlineHook
		resp, _ = resp.Next(ctx, &nextInlineHookSet)
		output = append(output, nextInlineHookSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}
