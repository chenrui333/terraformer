// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"

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
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := listAllVultrResources(context.Background(), client.BlockStorage.List)
	if err != nil {
		return fmt.Errorf("list vultr block storage: %w", err)
	}
	g.Resources = g.createResources(output)
	return nil
}
