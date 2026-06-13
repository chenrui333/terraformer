// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"

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
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := listAllVultrResources(context.Background(), client.ReservedIP.List)
	if err != nil {
		return fmt.Errorf("list vultr reserved IPs: %w", err)
	}
	g.Resources = g.createResources(output)
	return nil
}
