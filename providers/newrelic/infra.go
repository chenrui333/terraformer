// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	newrelic "github.com/newrelic/newrelic-client-go/newrelic"
)

type InfraGenerator struct {
	NewRelicService
}

func (g *InfraGenerator) createAlertInfraConditionResources(client *newrelic.NewRelic) error {
	alertPolicies, err := client.Alerts.ListPolicies(nil)
	if err != nil {
		return err
	}

	for _, alertPolicy := range alertPolicies {
		alertInfraConditions, err := client.Alerts.ListInfrastructureConditions(alertPolicy.ID)
		if err != nil {
			return err
		}
		for _, alertInfraCondition := range alertInfraConditions {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				fmt.Sprintf("%d:%d", alertPolicy.ID, alertInfraCondition.ID),
				fmt.Sprintf("%s-%d", normalizeResourceName(alertInfraCondition.Name), alertInfraCondition.ID),
				"newrelic_infra_alert_condition",
				g.ProviderName,
				map[string]string{
					"type": alertInfraCondition.Type,
				},
				[]string{},
				map[string]interface{}{}))
		}
	}
	return nil
}

func (g *InfraGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	err = g.createAlertInfraConditionResources(client)
	if err != nil {
		return err
	}

	return nil
}
