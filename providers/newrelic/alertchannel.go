// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	newrelic "github.com/newrelic/newrelic-client-go/v2/newrelic"
)

type AlertChannelGenerator struct {
	NewRelicService
}

func (g *AlertChannelGenerator) createAlertChannelResources(client *newrelic.NewRelic) error {
	alertChannels, err := client.Alerts.ListChannels()
	if err != nil {
		return err
	}

	for _, channel := range alertChannels {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			fmt.Sprintf("%d", channel.ID),
			fmt.Sprintf("%s-%d", normalizeResourceName(channel.Name), channel.ID),
			"newrelic_alert_channel",
			g.ProviderName,
			[]string{},
		))
	}

	return nil
}

func (g *AlertChannelGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	err = g.createAlertChannelResources(client)
	if err != nil {
		return err
	}

	return nil
}
