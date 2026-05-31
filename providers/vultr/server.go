// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type ServerGenerator struct {
	VultrService
}

func (g ServerGenerator) createResources(serverList []govultr.Instance) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, server := range serverList {
		resources = append(resources, terraformutils.NewSimpleResource(
			server.ID,
			server.ID,
			"vultr_server",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *ServerGenerator) InitResources() error {
	client := g.generateClient()
	output, _, _, err := client.Instance.List(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
