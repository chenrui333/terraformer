// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type StoreGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *StoreGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	stores, err := client.Project().Stores().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, store := range stores.Results {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			store.ID,
			store.Key,
			"commercetools_store",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
