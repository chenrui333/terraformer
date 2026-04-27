// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type BlockStorageGenerator struct {
	VultrService
}

func (g BlockStorageGenerator) createResources(blockStorageList []govultr.BlockStorage) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, blockStorage := range blockStorageList {
		resources = append(resources, terraformutils.NewSimpleResource(
			blockStorage.BlockStorageID,
			blockStorage.BlockStorageID,
			"vultr_block_storage",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *BlockStorageGenerator) InitResources() error {
	client := g.generateClient()
	output, err := client.BlockStorage.List(context.Background())
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
