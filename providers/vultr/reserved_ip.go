// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type ReservedIPGenerator struct {
	VultrService
}

func (g ReservedIPGenerator) createResources(ipList []govultr.ReservedIP) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, ip := range ipList {
		resources = append(resources, terraformutils.NewSimpleResource(
			ip.ReservedIPID,
			ip.ReservedIPID,
			"vultr_reserved_ip",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *ReservedIPGenerator) InitResources() error {
	client := g.generateClient()
	output, err := client.ReservedIP.List(context.Background())
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
