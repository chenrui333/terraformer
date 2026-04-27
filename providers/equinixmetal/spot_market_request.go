// SPDX-License-Identifier: Apache-2.0

package equinixmetal

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/packethost/packngo"
)

type SpotMarketRequestGenerator struct {
	EquinixMetalService
}

func (g SpotMarketRequestGenerator) listSpotMarketRequests(client *packngo.Client) ([]packngo.SpotMarketRequest, error) {
	spotMarketRequests, _, err := client.SpotMarketRequests.List(g.GetArgs()["project_id"].(string), nil)
	if err != nil {
		return nil, err
	}

	return spotMarketRequests, nil
}

func (g SpotMarketRequestGenerator) createResources(spotMarketRequestsList []packngo.SpotMarketRequest) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, spotMarketRequests := range spotMarketRequestsList {
		resources = append(resources, terraformutils.NewSimpleResource(
			spotMarketRequests.ID,
			spotMarketRequests.ID,
			"metal_spot_market_request",
			"equinixmetal",
			[]string{}))
	}
	return resources
}

func (g *SpotMarketRequestGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listSpotMarketRequests(client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
