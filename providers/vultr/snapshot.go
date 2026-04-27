// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type SnapshotGenerator struct {
	VultrService
}

func (g SnapshotGenerator) createResources(snapshotList []govultr.Snapshot) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, snapshot := range snapshotList {
		resources = append(resources, terraformutils.NewSimpleResource(
			snapshot.SnapshotID,
			snapshot.SnapshotID,
			"vultr_snapshot",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *SnapshotGenerator) InitResources() error {
	client := g.generateClient()
	output, err := client.Snapshot.List(context.Background())
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
