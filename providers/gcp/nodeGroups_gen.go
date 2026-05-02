// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var nodeGroupsAllowEmptyValues = []string{""}

var nodeGroupsAdditionalFields = map[string]interface{}{}

type NodeGroupsGenerator struct {
	GCPService
}

// Run on nodeGroupsList and create for each TerraformResource
func (g NodeGroupsGenerator) createResources(ctx context.Context, nodeGroupsList *compute.NodeGroupsListCall, zone string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := nodeGroupsList.Pages(ctx, func(page *compute.NodeGroupList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				zone+"/"+obj.Name,
				zone+"/"+obj.Name,
				"google_compute_node_group",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
					"zone":    zone,
				},
				nodeGroupsAllowEmptyValues,
				nodeGroupsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list nodeGroups: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each nodeGroups create 1 TerraformResource
// Need nodeGroups name as ID for terraform resource
func (g *NodeGroupsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	for _, zoneLink := range g.GetArgs()["region"].(compute.Region).Zones {
		t := strings.Split(zoneLink, "/")
		zone := t[len(t)-1]
		nodeGroupsList := computeService.NodeGroups.List(g.GetArgs()["project"].(string), zone)
		resources, err := g.createResources(ctx, nodeGroupsList, zone)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	return nil

}
