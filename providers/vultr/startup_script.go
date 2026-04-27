// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type StartupScriptGenerator struct {
	VultrService
}

func (g StartupScriptGenerator) createResources(scriptList []govultr.StartupScript) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, script := range scriptList {
		resources = append(resources, terraformutils.NewSimpleResource(
			script.ScriptID,
			script.ScriptID,
			"vultr_startup_script",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *StartupScriptGenerator) InitResources() error {
	client := g.generateClient()
	output, err := client.StartupScript.List(context.Background())
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
