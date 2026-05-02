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

func (g *WebhookGenerator) loadWebhooks(ctx context.Context, client *ldapi.APIClient) error {
	webhooks, _, err := client.WebhooksApi.GetAllWebhooks(ctx).Execute()
	if err != nil {
		return err
	}
	if webhooks == nil {
		return nil
	}
	for _, webhook := range webhooks.Items {
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
	return nil
}

func (g *WebhookGenerator) InitResources() error {
	return g.loadWebhooks(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient))
}
