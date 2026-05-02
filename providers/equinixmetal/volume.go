// SPDX-License-Identifier: Apache-2.0

package equinixmetal

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/packethost/packngo"
)

type VolumeGenerator struct {
	EquinixMetalService
}

func (g VolumeGenerator) listVolumes(client *packngo.Client) ([]packngo.Volume, error) {
	volumes, _, err := client.Volumes.List(g.GetArgs()["project_id"].(string), nil)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}

func (g VolumeGenerator) createResources(volumeList []packngo.Volume) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, volume := range volumeList {
		resources = append(resources, terraformutils.NewSimpleResource(
			volume.ID,
			volume.Name,
			"metal_volume",
			"equinixmetal",
			[]string{}))
	}
	return resources
}

func (g *VolumeGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := g.listVolumes(client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
