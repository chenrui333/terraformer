// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

type StackScriptGenerator struct {
	LinodeService
}

func (g StackScriptGenerator) createResources(stackscriptList []linodego.Stackscript) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, stackscript := range stackscriptList {
		// Avoid importing all community stackscripts
		if !stackscript.IsPublic {
			resources = append(resources, terraformutils.NewSimpleResource(
				strconv.Itoa(stackscript.ID),
				strconv.Itoa(stackscript.ID),
				"linode_stackscript",
				"linode",
				[]string{}))
		}
	}
	return resources
}

func (g *StackScriptGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := client.ListStackscripts(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
