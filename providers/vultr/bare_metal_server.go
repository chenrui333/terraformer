// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type BareMetalServerGenerator struct {
	VultrService
}

func (g BareMetalServerGenerator) createResources(serverList []govultr.BareMetalServer) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, server := range serverList {
		resources = append(resources, terraformutils.NewSimpleResource(
			server.ID,
			server.ID,
			"vultr_bare_metal_server",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *BareMetalServerGenerator) InitResources() error {
	client := g.generateClient()
	output, _, _, err := client.BareMetalServer.List(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
