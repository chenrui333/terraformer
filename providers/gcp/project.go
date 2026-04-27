// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"github.com/chenrui333/terraformer/terraformutils"
)

var projectAllowEmptyValues = []string{""}

var projectAdditionalFields = map[string]interface{}{}

type ProjectGenerator struct {
	GCPService
}

// Generate TerraformResources from GCP API,
func (g *ProjectGenerator) InitResources() error {
	g.Resources = append(g.Resources, terraformutils.NewResource(
		g.GetArgs()["project"].(string),
		g.GetArgs()["project"].(string),
		"google_project",
		g.ProviderName,
		map[string]string{
			"auto_create_network": "true",
		},
		projectAllowEmptyValues,
		projectAdditionalFields,
	))

	return nil
}
