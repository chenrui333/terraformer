// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type SSHKeyGenerator struct {
	VultrService
}

func (g SSHKeyGenerator) createResources(keyList []govultr.SSHKey) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, key := range keyList {
		resources = append(resources, terraformutils.NewSimpleResource(
			key.ID,
			key.ID,
			"vultr_ssh_key",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *SSHKeyGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := listAllVultrResources(context.Background(), client.SSHKey.List)
	if err != nil {
		return fmt.Errorf("list vultr SSH keys: %w", err)
	}
	g.Resources = g.createResources(output)
	return nil
}
