// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

type InstanceGenerator struct {
	LinodeService
}

func (g InstanceGenerator) createResources(instanceList []linodego.Instance) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, instance := range instanceList {
		resources = append(resources, terraformutils.NewSimpleResource(
			strconv.Itoa(instance.ID),
			strconv.Itoa(instance.ID),
			"linode_instance",
			"linode",
			[]string{}))
	}
	return resources
}

func (g *InstanceGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := client.ListInstances(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
