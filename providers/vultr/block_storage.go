// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type BlockStorageGenerator struct {
	VultrService
}

func (g BlockStorageGenerator) createResources(blockStorageList []govultr.BlockStorage) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, blockStorage := range blockStorageList {
		resources = append(resources, terraformutils.NewSimpleResource(
			blockStorage.ID,
			blockStorage.ID,
			"vultr_block_storage",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *BlockStorageGenerator) InitResources() error {
	client := g.generateClient()
	output, _, _, err := client.BlockStorage.List(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
