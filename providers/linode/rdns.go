// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

type RDNSGenerator struct {
	LinodeService
}

func (g RDNSGenerator) createResources(instanceIPList []linodego.InstanceIP) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, instanceIP := range instanceIPList {
		resources = append(resources, terraformutils.NewSimpleResource(
			instanceIP.Address,
			instanceIP.Address,
			"linode_rdns",
			"linode",
			[]string{}))
	}
	return resources
}

func (g *RDNSGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := client.ListIPAddresses(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
