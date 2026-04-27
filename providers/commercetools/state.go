// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type StateGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *StateGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	states, err := client.Project().States().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, state := range states.Results {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			state.ID,
			state.Key,
			"commercetools_state",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
