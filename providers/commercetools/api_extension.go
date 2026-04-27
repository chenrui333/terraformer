// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type APIExtensionGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *APIExtensionGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	extensions, err := client.Project().Extensions().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, extension := range extensions.Results {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			extension.ID,
			stringValue(extension.Key),
			"commercetools_api_extension",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
