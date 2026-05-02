// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type WebhookGenerator struct {
	LaunchDarklyService
}

func (g *WebhookGenerator) loadWebhooks(ctx context.Context, apiKey string) error {
	for path := "/webhooks"; path != ""; {
		webhooks := &ldapi.Webhooks{}
		if err := getLaunchDarklyAPI(ctx, apiKey, path, webhooks); err != nil {
			return err
		}

		for _, webhook := range webhooks.GetItems() {
			resource := terraformutils.NewResource(
				webhook.Id,
				resourceName(webhook.GetName(), webhook.Id),
				"launchdarkly_webhook",
				"launchdarkly",
				map[string]string{},
				[]string{},
				map[string]interface{}{})
			g.Resources = append(g.Resources, resource)
		}
		path = nextPagePath(webhooks.GetLinks())
	}
	return nil
}

func (g *WebhookGenerator) InitResources() error {
	return g.loadWebhooks(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["api_key"].(string))
}
