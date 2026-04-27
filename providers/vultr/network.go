// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type NetworkGenerator struct {
	VultrService
}

func (g NetworkGenerator) createResources(networkList []govultr.Network) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, network := range networkList {
		resources = append(resources, terraformutils.NewSimpleResource(
			network.NetworkID,
			network.NetworkID,
			"vultr_network",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *NetworkGenerator) InitResources() error {
	client := g.generateClient()
	output, err := client.Network.List(context.Background())
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
