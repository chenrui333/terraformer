// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
)

// the default node pool has a custom tag added by terraform
const digitaloceanKubernetesDefaultNodePoolTag = "terraform:default-node-pool"

type KubernetesClusterGenerator struct {
	DigitalOceanService
}

func (g *KubernetesClusterGenerator) loadKubernetesClusters(ctx context.Context, client *godo.Client) ([]*godo.KubernetesCluster, error) {
	list := []*godo.KubernetesCluster{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		clusters, resp, err := client.Kubernetes.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		for _, cluster := range clusters {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				cluster.ID,
				cluster.Name,
				"digitalocean_kubernetes_cluster",
				"digitalocean",
				[]string{}))
			list = append(list, cluster)
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	return list, nil
}

func (g *KubernetesClusterGenerator) loadKubernetesNodePools(cluster *godo.KubernetesCluster) {
	for _, nodePool := range cluster.NodePools {
		isDefaultPool := false
		for _, tag := range nodePool.Tags {
			if tag == digitaloceanKubernetesDefaultNodePoolTag {
				isDefaultPool = true
				break
			}
		}

		// skip default node pool since it is included in the digitalocean_kubernetes_cluster resource
		if !isDefaultPool {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				nodePool.ID,
				nodePool.Name,
				"digitalocean_kubernetes_node_pool",
				"digitalocean",
				map[string]string{"cluster_id": cluster.ID},
				[]string{},
				map[string]interface{}{}))
		}
	}
}

func (g *KubernetesClusterGenerator) InitResources() error {
	client := g.generateClient()
	clusters, err := g.loadKubernetesClusters(context.TODO(), client)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		g.loadKubernetesNodePools(cluster)
	}
	return nil
}
