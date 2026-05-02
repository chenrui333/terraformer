// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type NotificationsGenerator struct {
	CloudflareService
}

func (g *NotificationsGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	accountID := account.Identifier

	policies, err := api.ListNotificationPolicies(ctx, accountID)
	if err != nil {
		return err
	}
	for _, policy := range policies.Result {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			policy.ID,
			cloudflareResourceName(accountID, policy.Name, policy.ID),
			"cloudflare_notification_policy",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		))
	}

	webhooks, err := api.ListNotificationWebhooks(ctx, accountID)
	if err != nil {
		return err
	}
	for _, webhook := range webhooks.Result {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			webhook.ID,
			cloudflareResourceName(accountID, webhook.Name, webhook.ID),
			"cloudflare_notification_policy_webhooks",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
