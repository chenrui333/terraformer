// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"

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
			"vultr_instance",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *ServerGenerator) initResources(client *govultr.Client) error {
	output, err := listAllVultrResources(context.Background(), client.Instance.List)
	if err != nil {
		return fmt.Errorf("list vultr instances: %w", err)
	}
	g.Resources = g.createResources(output)
	return nil
}

func (g *ServerGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	return g.initResources(client)
}
