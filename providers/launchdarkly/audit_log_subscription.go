// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/iancoleman/strcase"
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
			subscriptionID,
			fmt.Sprintf("%s-%s", integrationKey, resourceName(subscription.GetName(), subscriptionID)),
			"launchdarkly_audit_log_subscription",
			"launchdarkly",
			auditLogSubscriptionAttributes(integrationKey, subscription.Config),
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func auditLogSubscriptionAttributes(integrationKey string, config map[string]interface{}) map[string]string {
	attributes := map[string]string{
		"integration_key": integrationKey,
		"config.%":        strconv.Itoa(len(config)),
	}
	for key, value := range config {
		attributes["config."+auditLogSubscriptionConfigKey(key)] = auditLogSubscriptionConfigValue(value)
	}
	return attributes
}

func auditLogSubscriptionConfigKey(key string) string {
	if key == "last9" {
		return key
	}
	return strcase.ToSnake(key)
}

func auditLogSubscriptionConfigValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	default:
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
		return fmt.Sprint(v)
	}
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
