// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type ShippingZoneGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *ShippingZoneGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	zones, err := client.Project().Zones().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, zone := range zones.Results {
		resourceName := stringValue(zone.Key)
		if resourceName == "" {
			resourceName = zone.Name
		}

		g.Resources = append(g.Resources, terraformutils.NewResource(
			zone.ID,
			resourceName,
			"commercetools_shipping_zone",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
