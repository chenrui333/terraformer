// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/mackerelio/mackerel-client-go"
)

// MonitorGenerator ...
type MonitorGenerator struct {
	MackerelService
}

func (g *MonitorGenerator) createResources(monitors []mackerel.Monitor) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, monitor := range monitors {
		resources = append(resources, g.createResource(monitor.MonitorID()))
	}
	return resources
}

func (g *MonitorGenerator) createResource(monitorID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		monitorID,
		fmt.Sprintf("monitor_%s", monitorID),
		"mackerel_monitor",
		"mackerel",
		[]string{},
	)
}

// InitResources Generate TerraformResources from Mackerel API,
// from each monitor create 1 TerraformResource.
// Need Monitor ID as ID for terraform resource
func (g *MonitorGenerator) InitResources() error {
	client := g.Args["mackerelClient"].(*mackerel.Client)
	monitors, err := client.FindMonitors()
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, g.createResources(monitors)...)
	return nil
}
