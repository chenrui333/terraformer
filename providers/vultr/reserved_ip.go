// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type ReservedIPGenerator struct {
	VultrService
}

func (g ReservedIPGenerator) createResources(ipList []govultr.ReservedIP) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, ip := range ipList {
		resources = append(resources, terraformutils.NewSimpleResource(
			ip.ID,
			ip.ID,
			"vultr_reserved_ip",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *ReservedIPGenerator) InitResources() error {
	client := g.generateClient()
	output, _, _, err := client.ReservedIP.List(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
