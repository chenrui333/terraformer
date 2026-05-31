// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type NetworkGenerator struct {
	VultrService
}

func (g NetworkGenerator) createResources(networkList []govultr.VPC) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, network := range networkList {
		resources = append(resources, terraformutils.NewSimpleResource(
			network.ID,
			network.ID,
			"vultr_network",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *NetworkGenerator) InitResources() error {
	client := g.generateClient()
	output, _, _, err := client.VPC.List(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
