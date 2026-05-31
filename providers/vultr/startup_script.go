// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type StartupScriptGenerator struct {
	VultrService
}

func (g StartupScriptGenerator) createResources(scriptList []govultr.StartupScript) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, script := range scriptList {
		resources = append(resources, terraformutils.NewSimpleResource(
			script.ID,
			script.ID,
			"vultr_startup_script",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *StartupScriptGenerator) InitResources() error {
	client := g.generateClient()
	output, _, _, err := client.StartupScript.List(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
