// Copyright 2019 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	"context"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kafka/types"
)

var mskAllowEmptyValues = []string{"tags."}

type MskGenerator struct {
	AWSService
}

// loadMskClusters loads provisioned MSK clusters (aws_msk_cluster)
func (g *MskGenerator) loadMskClusters(svc *kafka.Client) error {
	p := kafka.NewListClustersPaginator(svc, &kafka.ListClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, clusterInfo := range page.ClusterInfoList {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(clusterInfo.ClusterArn),
				StringValue(clusterInfo.ClusterName),
				"aws_msk_cluster",
				"aws",
				mskAllowEmptyValues,
			))
		}
	}
	return nil
}

// loadMskConfigurations loads MSK configurations (aws_msk_configuration)
// Import ID: configuration ARN
func (g *MskGenerator) loadMskConfigurations(svc *kafka.Client) error {
	p := kafka.NewListConfigurationsPaginator(svc, &kafka.ListConfigurationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, config := range page.Configurations {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(config.Arn),
				StringValue(config.Name),
				"aws_msk_configuration",
				"aws",
				mskAllowEmptyValues,
			))
		}
	}
	return nil
}

// loadMskReplicators loads MSK replicators (aws_msk_replicator)
// Import ID: replicator ARN
func (g *MskGenerator) loadMskReplicators(svc *kafka.Client) error {
	p := kafka.NewListReplicatorsPaginator(svc, &kafka.ListReplicatorsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, replicator := range page.Replicators {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(replicator.ReplicatorArn),
				StringValue(replicator.ReplicatorName),
				"aws_msk_replicator",
				"aws",
				mskAllowEmptyValues,
			))
		}
	}
	return nil
}

// loadMskVpcConnections loads MSK VPC connections (aws_msk_vpc_connection)
// Import ID: VPC connection ARN
func (g *MskGenerator) loadMskVpcConnections(svc *kafka.Client) error {
	p := kafka.NewListVpcConnectionsPaginator(svc, &kafka.ListVpcConnectionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, vpcConn := range page.VpcConnections {
			// Generate a deterministic resource name from the VPC connection ARN
			// ARN format: arn:aws:kafka:region:account:vpc-connection/account/name/uuid
			resourceName := StringValue(vpcConn.VpcConnectionArn)
			if parts := strings.Split(resourceName, "/"); len(parts) >= 3 {
				resourceName = parts[len(parts)-2] + "-" + parts[len(parts)-1]
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(vpcConn.VpcConnectionArn),
				resourceName,
				"aws_msk_vpc_connection",
				"aws",
				mskAllowEmptyValues,
			))
		}
	}
	return nil
}

// loadMskServerlessClusters loads MSK serverless clusters (aws_msk_serverless_cluster)
// Import ID: serverless cluster ARN
func (g *MskGenerator) loadMskServerlessClusters(svc *kafka.Client) error {
	// Use ListClustersV2 with filter for serverless clusters
	serverlessFilter := string(types.ClusterTypeServerless)
	p := kafka.NewListClustersV2Paginator(svc, &kafka.ListClustersV2Input{
		ClusterTypeFilter: &serverlessFilter,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.ClusterInfoList {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(cluster.ClusterArn),
				StringValue(cluster.ClusterName),
				"aws_msk_serverless_cluster",
				"aws",
				mskAllowEmptyValues,
			))
		}
	}
	return nil
}

// loadMskClusterPolicies loads cluster policies for each MSK cluster (aws_msk_cluster_policy)
// Import ID: cluster ARN
func (g *MskGenerator) loadMskClusterPolicies(svc *kafka.Client) error {
	// First, get all clusters (both provisioned and serverless)
	clusterArns := []string{}

	// Get provisioned clusters
	p := kafka.NewListClustersPaginator(svc, &kafka.ListClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.ClusterInfoList {
			clusterArns = append(clusterArns, StringValue(cluster.ClusterArn))
		}
	}

	// Get serverless clusters
	serverlessFilter := string(types.ClusterTypeServerless)
	pv2 := kafka.NewListClustersV2Paginator(svc, &kafka.ListClustersV2Input{
		ClusterTypeFilter: &serverlessFilter,
	})
	for pv2.HasMorePages() {
		page, err := pv2.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.ClusterInfoList {
			clusterArns = append(clusterArns, StringValue(cluster.ClusterArn))
		}
	}

	// Sort for deterministic ordering
	sort.Strings(clusterArns)

	// For each cluster, check if it has a policy
	for _, clusterArn := range clusterArns {
		_, err := svc.GetClusterPolicy(context.TODO(), &kafka.GetClusterPolicyInput{
			ClusterArn: &clusterArn,
		})
		if err != nil {
			// If no policy exists, the API returns an error - skip this cluster
			// Common error: NotFoundException when no policy is attached
			continue
		}

		// Extract cluster name from ARN for resource naming
		// ARN format: arn:aws:kafka:region:account:cluster/name/uuid
		resourceName := clusterArn
		if parts := strings.Split(clusterArn, "/"); len(parts) >= 2 {
			resourceName = parts[1] + "-policy"
		}

		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			clusterArn,
			resourceName,
			"aws_msk_cluster_policy",
			"aws",
			mskAllowEmptyValues,
		))
	}

	return nil
}

// loadMskScramSecretAssociations loads SCRAM secret associations for each MSK cluster
// aws_msk_scram_secret_association import ID: cluster ARN
// aws_msk_single_scram_secret_association import ID: cluster_arn,secret_arn
func (g *MskGenerator) loadMskScramSecretAssociations(svc *kafka.Client) error {
	// Get provisioned clusters (SCRAM is only supported on provisioned clusters)
	p := kafka.NewListClustersPaginator(svc, &kafka.ListClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.ClusterInfoList {
			clusterArn := StringValue(cluster.ClusterArn)
			clusterName := StringValue(cluster.ClusterName)

			// List SCRAM secrets for this cluster
			secretArns := []string{}
			sp := kafka.NewListScramSecretsPaginator(svc, &kafka.ListScramSecretsInput{
				ClusterArn: &clusterArn,
			})
			for sp.HasMorePages() {
				secretPage, err := sp.NextPage(context.TODO())
				if err != nil {
					// If listing fails (e.g., SCRAM not enabled), skip this cluster
					break
				}
				secretArns = append(secretArns, secretPage.SecretArnList...)
			}

			if len(secretArns) == 0 {
				continue
			}

			// Sort for deterministic ordering
			sort.Strings(secretArns)

			// Create aws_msk_scram_secret_association resource (one per cluster)
			// This resource manages all secrets for a cluster
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				clusterArn,
				clusterName+"-scram-secrets",
				"aws_msk_scram_secret_association",
				"aws",
				mskAllowEmptyValues,
			))

			// Create aws_msk_single_scram_secret_association resources (one per secret)
			for _, secretArn := range secretArns {
				// Import ID format: cluster_arn,secret_arn
				importID := clusterArn + "," + secretArn

				// Extract secret name from ARN for resource naming
				// ARN format: arn:aws:secretsmanager:region:account:secret:name-suffix
				secretName := secretArn
				if parts := strings.Split(secretArn, ":"); len(parts) >= 7 {
					secretName = parts[6]
					// Remove the random suffix if present (e.g., "mysecret-AbCdEf")
					if idx := strings.LastIndex(secretName, "-"); idx > 0 && len(secretName)-idx <= 7 {
						secretName = secretName[:idx]
					}
				}

				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					importID,
					clusterName+"-"+secretName,
					"aws_msk_single_scram_secret_association",
					"aws",
					mskAllowEmptyValues,
				))
			}
		}
	}

	return nil
}

func (g *MskGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := kafka.NewFromConfig(config)

	// Load provisioned clusters
	if err := g.loadMskClusters(svc); err != nil {
		return err
	}

	// Load MSK configurations
	if err := g.loadMskConfigurations(svc); err != nil {
		return err
	}

	// Load MSK replicators
	if err := g.loadMskReplicators(svc); err != nil {
		return err
	}

	// Load MSK VPC connections
	if err := g.loadMskVpcConnections(svc); err != nil {
		return err
	}

	// Load serverless clusters
	if err := g.loadMskServerlessClusters(svc); err != nil {
		return err
	}

	// Load cluster policies
	if err := g.loadMskClusterPolicies(svc); err != nil {
		return err
	}

	// Load SCRAM secret associations
	if err := g.loadMskScramSecretAssociations(svc); err != nil {
		return err
	}

	return nil
}

func (g *MskGenerator) PostConvertHook() error {
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_msk_cluster" {
			continue
		}
		if r.InstanceState.Attributes["configuration_info.0.revision"] == "0" {
			delete(r.Item, "configuration_info")
		}
	}
	return nil
}
