// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

type SSHKeyGenerator struct {
	LinodeService
}

func (g SSHKeyGenerator) createResources(keyList []linodego.SSHKey) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, key := range keyList {
		resources = append(resources, terraformutils.NewSimpleResource(
			strconv.Itoa(key.ID),
			strconv.Itoa(key.ID),
			"linode_sshkey",
			"linode",
			[]string{}))
	}
	return resources
}

func (g *SSHKeyGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := client.ListSSHKeys(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
