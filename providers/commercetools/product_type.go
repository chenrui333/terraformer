// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type ProductTypeGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *ProductTypeGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	productTypes, err := client.Project().ProductTypes().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, productType := range productTypes.Results {
		resourceName := stringValue(productType.Key)
		if resourceName == "" {
			resourceName = normalizeResourceName(productType.Name)
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			productType.ID,
			resourceName,
			"commercetools_product_type",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
