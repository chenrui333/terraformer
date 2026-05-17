// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"log"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// WebhookAllowEmptyValues ...
	WebhookAllowEmptyValues = []string{"payload"}
)

// WebhookGenerator ...
type WebhookGenerator struct {
	DatadogService
}

func (g *WebhookGenerator) createResource(webhook datadogV1.WebhooksIntegration) (terraformutils.Resource, error) {
	webhookName := webhook.GetName()
	if webhookName == "" {
		return terraformutils.Resource{}, fmt.Errorf("webhook missing name")
	}

	return terraformutils.NewSimpleResource(
		webhookName,
		fmt.Sprintf("webhook_%s", webhookName),
		"datadog_webhook",
		"datadog",
		WebhookAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API.
// The Datadog webhooks API supports get-by-name but does not expose a list endpoint,
// so this generator requires an ID filter containing the webhook name.
func (g *WebhookGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV1.NewWebhooksIntegrationApi(datadogClient)

	resources := []terraformutils.Resource{}
	matchedIDFilter := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" {
			continue
		}
		if !filter.IsApplicable("webhook") {
			continue
		}
		matchedIDFilter = true
		for _, value := range filter.AcceptableValues {
			webhook, httpResp, err := api.GetWebhooksIntegration(auth, value)
			closeDatadogResponseBody(httpResp)
			if err != nil {
				return err
			}
			resource, err := g.createResource(webhook)
			if err != nil {
				return err
			}
			resources = append(resources, resource)
		}
	}

	if matchedIDFilter {
		g.Resources = resources
		return nil
	}

	log.Print("Filter(resource id) is required to import datadog_webhook resources because the Datadog API does not provide a webhook list endpoint")
	return nil
}
