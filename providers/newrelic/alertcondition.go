// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	newrelic "github.com/newrelic/newrelic-client-go/v2/newrelic"
)

type AlertConditionGenerator struct {
	NewRelicService
}

func (g *AlertConditionGenerator) createAlertConditionResources(client *newrelic.NewRelic) error {
	alertPolicies, err := client.Alerts.ListPolicies(nil)
	if err != nil {
		return err
	}

	for _, alertPolicy := range alertPolicies {
		alertConditions, err := client.Alerts.ListConditions(alertPolicy.ID)
		if err != nil {
			return err
		}

		for _, alertCondition := range alertConditions {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%d:%d", alertPolicy.ID, alertCondition.ID),
				fmt.Sprintf("%s-%d", normalizeResourceName(alertCondition.Name), alertCondition.ID),
				"newrelic_alert_condition",
				g.ProviderName,
				[]string{}))
		}
	}
	return nil
}

func (g *AlertConditionGenerator) createAlertNrqlConditionResources(client *newrelic.NewRelic) error {
	alertPolicies, err := client.Alerts.ListPolicies(nil)
	if err != nil {
		return err
	}

	for _, alertPolicy := range alertPolicies {
		nrqlConditions, err := client.Alerts.ListNrqlConditions(alertPolicy.ID)
		if err != nil {
			return err
		}

		for _, nrqlCondition := range nrqlConditions {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%d:%d", alertPolicy.ID, nrqlCondition.ID),
				fmt.Sprintf("%s-%d", normalizeResourceName(nrqlCondition.Name), nrqlCondition.ID),
				"newrelic_nrql_alert_condition",
				g.ProviderName,
				[]string{}))
		}
	}
	return nil
}

func (g *AlertConditionGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	funcs := []func(*newrelic.NewRelic) error{
		g.createAlertConditionResources,
		g.createAlertNrqlConditionResources,
	}

	for _, f := range funcs {
		err := f(client)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *AlertConditionGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type == "newrelic_alert_condition" {
			if resource.Item["violation_close_timer"] == "0" {
				delete(g.Resources[i].Item, "violation_close_timer")
			}
		}
	}

	return nil
}
