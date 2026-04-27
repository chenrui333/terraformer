// SPDX-License-Identifier: Apache-2.0

package equinixmetal

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/packethost/packngo"
)

type SSHKeyGenerator struct {
	EquinixMetalService
}

func (g SSHKeyGenerator) listSSHKeys(client *packngo.Client) ([]packngo.SSHKey, error) {
	sshKeys, _, err := client.SSHKeys.List()
	if err != nil {
		return nil, err
	}

	return sshKeys, nil
}

func (g SSHKeyGenerator) createResources(sshLeyList []packngo.SSHKey) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, sshKey := range sshLeyList {
		resources = append(resources, terraformutils.NewSimpleResource(
			sshKey.ID,
			sshKey.Label,
			"metal_ssh_key",
			"equinixmetal",
			[]string{}))
	}
	return resources
}

func (g *SSHKeyGenerator) InitResources() error {
	client := g.generateClient()
	output, err := g.listSSHKeys(client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
