// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

type ImageGenerator struct {
	LinodeService
}

func (g ImageGenerator) createResources(imageList []linodego.Image) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, image := range imageList {
		resources = append(resources, terraformutils.NewSimpleResource(
			image.ID,
			image.ID,
			"linode_image",
			"linode",
			[]string{}))
	}
	return resources
}

func (g *ImageGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := client.ListImages(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
