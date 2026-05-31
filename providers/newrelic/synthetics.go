// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	newrelic "github.com/newrelic/newrelic-client-go/v2/newrelic"
)

type SyntheticsGenerator struct {
	NewRelicService
}

func (g *SyntheticsGenerator) createSyntheticsMonitorResources(client *newrelic.NewRelic) error {
	allMonitors, err := client.Synthetics.ListMonitors()
	if err != nil {
		return err
	}

	for _, monitor := range allMonitors {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			fmt.Sprint(monitor.ID),
			fmt.Sprintf("%s-%s", normalizeResourceName(monitor.Name), monitor.ID),
			"newrelic_synthetics_monitor",
			g.ProviderName,
			[]string{}))
	}

	return nil
}

func (g *SyntheticsGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	err = g.createSyntheticsMonitorResources(client)
	if err != nil {
		return err
	}

	return nil
}
