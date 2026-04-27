// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type SSHKeyGenerator struct {
	VultrService
}

func (g SSHKeyGenerator) createResources(keyList []govultr.SSHKey) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, key := range keyList {
		resources = append(resources, terraformutils.NewSimpleResource(
			key.SSHKeyID,
			key.SSHKeyID,
			"vultr_ssh_key",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *SSHKeyGenerator) InitResources() error {
	client := g.generateClient()
	output, err := client.SSHKey.List(context.Background())
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
