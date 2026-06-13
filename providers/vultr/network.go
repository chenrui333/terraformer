// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"

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
			"vultr_vpc",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *NetworkGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := listAllVultrResources(context.Background(), client.VPC.List)
	if err != nil {
		return fmt.Errorf("list vultr VPCs: %w", err)
	}
	g.Resources = g.createResources(output)
	return nil
}
