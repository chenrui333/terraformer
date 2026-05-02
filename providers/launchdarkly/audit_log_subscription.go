// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

var auditLogSubscriptionIntegrationKeys = []string{
	"chronosphere",
	"cloudtrail",
	"datadog",
	"dynatrace",
	"dynatrace-v2",
	"elastic",
	"grafana",
	"honeycomb",
	"jira",
	"kosli",
	"last9",
	"logdna",
	"msteams",
	"new-relic-apm",
	"pagerduty",
	"signalfx",
	"slack",
	"splunk",
}

type AuditLogSubscriptionGenerator struct {
	LaunchDarklyService
}

func (g *AuditLogSubscriptionGenerator) loadAuditLogSubscriptions(ctx context.Context, client *ldapi.APIClient, integrationKey string) error {
	subscriptions, resp, err := client.IntegrationAuditLogSubscriptionsApi.GetSubscriptions(ctx, integrationKey).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	if subscriptions == nil {
		return nil
	}
	for _, subscription := range subscriptions.GetItems() {
		subscriptionID := subscription.GetId()
		resource := terraformutils.NewResource(
			fmt.Sprintf("%s/%s", integrationKey, subscriptionID),
			resourceName(subscription.GetName(), subscriptionID),
			"launchdarkly_audit_log_subscription",
			"launchdarkly",
			map[string]string{
				"integration_key": integrationKey,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *AuditLogSubscriptionGenerator) InitResources() error {
	ctx := g.GetArgs()["ctx"].(context.Context)
	client := g.GetArgs()["client"].(*ldapi.APIClient)

	for _, integrationKey := range auditLogSubscriptionIntegrationKeys {
		if err := g.loadAuditLogSubscriptions(ctx, client, integrationKey); err != nil {
			return err
		}
	}
	return nil
}
