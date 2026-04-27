// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type ShippingMethodGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *ShippingMethodGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	zones, err := client.Project().ShippingMethods().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, zone := range zones.Results {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			zone.ID,
			stringValue(zone.Key),
			"commercetools_shipping_method",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
