// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type CustomObjectGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *CustomObjectGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	customObjects, err := client.Project().CustomObjects().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, customObject := range customObjects.Results {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			customObject.ID,
			customObject.Key,
			"commercetools_custom_object",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
