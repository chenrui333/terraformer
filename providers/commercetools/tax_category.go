// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type TaxCategoryGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *TaxCategoryGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	categories, err := client.Project().TaxCategories().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, category := range categories.Results {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			category.ID,
			stringValue(category.Key),
			"commercetools_tax_category",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
