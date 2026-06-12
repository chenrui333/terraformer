// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

type VolumeGenerator struct {
	LinodeService
}

func (g VolumeGenerator) createResources(volumeList []linodego.Volume) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, volume := range volumeList {
		resources = append(resources, terraformutils.NewSimpleResource(
			strconv.Itoa(volume.ID),
			strconv.Itoa(volume.ID),
			"linode_volume",
			"linode",
			[]string{}))
	}
	return resources
}

func (g *VolumeGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := client.ListVolumes(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
