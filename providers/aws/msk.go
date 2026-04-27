// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var mskAllowEmptyValues = []string{"tags."}

type MskGenerator struct {
	AWSService
}

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

func (g *MskGenerator) loadMskVpcConnections(svc *kafka.Client) error {
	p := kafka.NewListVpcConnectionsPaginator(svc, &kafka.ListVpcConnectionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, vpcConn := range page.VpcConnections {
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

func (g *MskGenerator) loadMskServerlessClusters(svc *kafka.Client) error {
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

func (g *MskGenerator) loadMskClusterPolicies(svc *kafka.Client) error {
	clusterArns := []string{}

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

	sort.Strings(clusterArns)

	for _, clusterArn := range clusterArns {
		_, err := svc.GetClusterPolicy(context.TODO(), &kafka.GetClusterPolicyInput{
			ClusterArn: &clusterArn,
		})
		if err != nil {
			continue
		}

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

func (g *MskGenerator) loadMskScramSecretAssociations(svc *kafka.Client) error {
	p := kafka.NewListClustersPaginator(svc, &kafka.ListClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.ClusterInfoList {
			clusterArn := StringValue(cluster.ClusterArn)
			clusterName := StringValue(cluster.ClusterName)

			secretArns := []string{}
			sp := kafka.NewListScramSecretsPaginator(svc, &kafka.ListScramSecretsInput{
				ClusterArn: &clusterArn,
			})
			for sp.HasMorePages() {
				secretPage, err := sp.NextPage(context.TODO())
				if err != nil {
					break
				}
				secretArns = append(secretArns, secretPage.SecretArnList...)
			}

			if len(secretArns) == 0 {
				continue
			}

			sort.Strings(secretArns)

			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				clusterArn,
				clusterName+"-scram-secrets",
				"aws_msk_scram_secret_association",
				"aws",
				mskAllowEmptyValues,
			))

			for _, secretArn := range secretArns {
				importID := clusterArn + "," + secretArn

				// ARN format: arn:aws:secretsmanager:region:account:secret:name-suffix
				secretName := secretArn
				if parts := strings.Split(secretArn, ":"); len(parts) >= 7 {
					secretName = parts[6]
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

	if err := g.loadMskClusters(svc); err != nil {
		return err
	}
	if err := g.loadMskConfigurations(svc); err != nil {
		return err
	}
	if err := g.loadMskReplicators(svc); err != nil {
		return err
	}
	if err := g.loadMskVpcConnections(svc); err != nil {
		return err
	}
	if err := g.loadMskServerlessClusters(svc); err != nil {
		return err
	}
	if err := g.loadMskClusterPolicies(svc); err != nil {
		return err
	}
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
