// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dataproc/v1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var dataprocAllowEmptyValues = []string{""}

var dataprocAdditionalFields = map[string]interface{}{}

type DataprocGenerator struct {
	GCPService
}

// Run on DataprocClusterList and create for each TerraformResource
func (g DataprocGenerator) createClusterResources(ctx context.Context, clusterList *dataproc.ProjectsRegionsClustersListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := clusterList.Pages(ctx, func(page *dataproc.ListClustersResponse) error {
		for _, cluster := range page.Clusters {
			resource := terraformutils.NewResource(
				cluster.ClusterName,
				cluster.ClusterName,
				"google_dataproc_cluster",
				g.ProviderName,
				map[string]string{
					"name":    cluster.ClusterName,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				dataprocAllowEmptyValues,
				dataprocAdditionalFields,
			)
			resource.IgnoreKeys = append(resource.IgnoreKeys, "^cluster_config.[0-9].delete_autogen_bucket$")
			resources = append(resources, resource)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list dataproc clusters: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each DataprocGenerator create 1 TerraformResource
// Need DataprocGenerator name as ID for terraform resource
func (g *DataprocGenerator) InitResources() error {
	ctx := context.Background()
	dataprocService, err := dataproc.NewService(ctx)
	if err != nil {
		return err
	}

	clusterList := dataprocService.Projects.Regions.Clusters.List(g.GetArgs()["project"].(string), g.GetArgs()["region"].(compute.Region).Name)
	resources, err := g.createClusterResources(ctx, clusterList)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil
}
