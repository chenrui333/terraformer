// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/mackerelio/mackerel-client-go"
)

// DowntimeGenerator ...
type DowntimeGenerator struct {
	MackerelService
}

func (g *DowntimeGenerator) createResources(downtimes []*mackerel.Downtime) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, downtime := range downtimes {
		resources = append(resources, g.createResource(downtime.ID))
	}
	return resources
}

func (g *DowntimeGenerator) createResource(downtimeID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		downtimeID,
		fmt.Sprintf("downtime_%s", downtimeID),
		"mackerel_downtime",
		"mackerel",
		[]string{},
	)
}

// InitResources Generate TerraformResources from Mackerel API,
// from each downtime create 1 TerraformResource.
// Need Downtime ID as ID for terraform resource
func (g *DowntimeGenerator) InitResources() error {
	client := g.Args["mackerelClient"].(*mackerel.Client)
	downtimes, err := client.FindDowntimes()
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, g.createResources(downtimes)...)
	return nil
}
