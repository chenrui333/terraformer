// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type TypesGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *TypesGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	types, err := client.Project().Types().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, customType := range types.Results {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			customType.ID,
			customType.Key,
			"commercetools_type",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
