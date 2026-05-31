// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	newrelic "github.com/newrelic/newrelic-client-go/v2/newrelic"
)

type AlertPolicyGenerator struct {
	NewRelicService
}

func (g *AlertPolicyGenerator) createAlertPolicyResources(client *newrelic.NewRelic) error {
	alertPolicies, err := client.Alerts.ListPolicies(nil)
	if err != nil {
		return err
	}

	for _, alertPolicy := range alertPolicies {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			fmt.Sprintf("%d", alertPolicy.ID),
			fmt.Sprintf("%s-%d", normalizeResourceName(alertPolicy.Name), alertPolicy.ID),
			"newrelic_alert_policy",
			g.ProviderName,
			[]string{}))
	}

	return nil
}

func (g *AlertPolicyGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	err = g.createAlertPolicyResources(client)
	if err != nil {
		return err
	}

	return nil
}
