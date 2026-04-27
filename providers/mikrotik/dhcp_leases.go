// SPDX-License-Identifier: Apache-2.0
package mikrotik

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/ddelnano/terraform-provider-mikrotik/client"
)

type DhcpLeaseGenerator struct {
	MikrotikService
}

func (g DhcpLeaseGenerator) createResources(leases []client.DhcpLease) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, lease := range leases {
		resourceName := lease.Id
		if lease.Hostname != "" {
			resourceName = fmt.Sprintf("%s-%s", lease.Hostname, lease.Id)
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			lease.Id,
			resourceName,
			"mikrotik_dhcp_lease",
			"mikrotik",
			[]string{}))
	}
	return resources
}

func (g *DhcpLeaseGenerator) InitResources() error {
	client := g.generateClient()
	leases, err := client.ListDhcpLeases()

	if err != nil {
		return err
	}
	g.Resources = g.createResources(leases)
	return nil
}
