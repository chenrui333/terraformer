// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

type NotificationsGenerator struct {
	CloudflareService
}

func listNotificationPolicies(ctx context.Context, api *cf.API, accountID string) ([]cf.NotificationPolicy, error) {
	policies := []cf.NotificationPolicy{}
	for page := 1; ; page++ {
		values := url.Values{}
		values.Set("page", strconv.Itoa(page))
		values.Set("per_page", strconv.Itoa(cloudflarePageSize))
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/alerting/v3/policies?%s", accountID, values.Encode()),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pagePolicies []cf.NotificationPolicy
		if err := json.Unmarshal(response.Result, &pagePolicies); err != nil {
			return nil, err
		}
		policies = append(policies, pagePolicies...)
		if response.ResultInfo == nil || !response.ResultInfo.HasMorePages() {
			break
		}
	}
	return policies, nil
}

func listNotificationWebhooks(ctx context.Context, api *cf.API, accountID string) ([]cf.NotificationWebhookIntegration, error) {
	webhooks := []cf.NotificationWebhookIntegration{}
	for page := 1; ; page++ {
		values := url.Values{}
		values.Set("page", strconv.Itoa(page))
		values.Set("per_page", strconv.Itoa(cloudflarePageSize))
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/alerting/v3/destinations/webhooks?%s", accountID, values.Encode()),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageWebhooks []cf.NotificationWebhookIntegration
		if err := json.Unmarshal(response.Result, &pageWebhooks); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, pageWebhooks...)
		if response.ResultInfo == nil || !response.ResultInfo.HasMorePages() {
			break
		}
	}
	return webhooks, nil
}

func (g *NotificationsGenerator) InitResources() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	accountID := account.Identifier

	policies, err := listNotificationPolicies(ctx, api, accountID)
	if err != nil {
		return err
	}
	for _, policy := range policies {
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

	webhooks, err := listNotificationWebhooks(ctx, api, accountID)
	if err != nil {
		return err
	}
	for _, webhook := range webhooks {
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
