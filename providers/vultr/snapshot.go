// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type SnapshotGenerator struct {
	VultrService
}

func (g SnapshotGenerator) createResources(snapshotList []govultr.Snapshot) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, snapshot := range snapshotList {
		resources = append(resources, terraformutils.NewSimpleResource(
			snapshot.ID,
			snapshot.ID,
			"vultr_snapshot",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *SnapshotGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := listAllVultrResources(context.Background(), client.Snapshot.List)
	if err != nil {
		return fmt.Errorf("list vultr snapshots: %w", err)
	}
	g.Resources = g.createResources(output)
	return nil
}
