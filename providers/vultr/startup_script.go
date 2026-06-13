// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"

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
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := listAllVultrResources(context.Background(), client.StartupScript.List)
	if err != nil {
		return fmt.Errorf("list vultr startup scripts: %w", err)
	}
	g.Resources = g.createResources(output)
	return nil
}
