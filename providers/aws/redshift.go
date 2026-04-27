// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"strings"

	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/redshift"
)

var RedshiftAllowEmptyValues = []string{"tags."}

type RedshiftGenerator struct {
	AWSService
}

func (g *RedshiftGenerator) loadClusters(svc *redshift.Client) error {
	p := redshift.NewDescribeClustersPaginator(svc, &redshift.DescribeClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, db := range page.Clusters {
			resourceName := StringValue(db.ClusterIdentifier)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_redshift_cluster",
				"aws",
				RedshiftAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RedshiftGenerator) loadParameterGroups(svc *redshift.Client) error {
	p := redshift.NewDescribeClusterParameterGroupsPaginator(svc, &redshift.DescribeClusterParameterGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, parameterGroup := range page.ParameterGroups {
			resourceName := StringValue(parameterGroup.ParameterGroupName)
			if strings.Contains(resourceName, ".") {
				continue // skip default Default ParameterGroups like default.mysql5.6
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_redshift_parameter_group",
				"aws",
				RedshiftAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RedshiftGenerator) loadSubnetGroups(svc *redshift.Client) error {
	p := redshift.NewDescribeClusterSubnetGroupsPaginator(svc, &redshift.DescribeClusterSubnetGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, subnet := range page.ClusterSubnetGroups {
			resourceName := StringValue(subnet.ClusterSubnetGroupName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_redshift_subnet_group",
				"aws",
				RedshiftAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RedshiftGenerator) loadEventSubscription(svc *redshift.Client) error {
	p := redshift.NewDescribeEventSubscriptionsPaginator(svc, &redshift.DescribeEventSubscriptionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, eventSubscription := range page.EventSubscriptionsList {
			resourceName := StringValue(eventSubscription.CustomerAwsId)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_redshift_event_subscription",
				"aws",
				RedshiftAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RedshiftGenerator) loadSnapshotSchedules(svc *redshift.Client) error {
	p := redshift.NewDescribeSnapshotSchedulesPaginator(svc, &redshift.DescribeSnapshotSchedulesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, snapshotSchedule := range page.SnapshotSchedules {
			resourceName := StringValue(snapshotSchedule.ScheduleIdentifier)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_redshift_snapshot_schedule",
				"aws",
				RedshiftAllowEmptyValues,
			))

			for _, associatedCluster := range snapshotSchedule.AssociatedClusters {
				clusterName := StringValue(associatedCluster.ClusterIdentifier)
				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					clusterName+"/"+resourceName,
					clusterName+"_"+resourceName,
					"aws_redshift_snapshot_schedule_association",
					"aws",
					RedshiftAllowEmptyValues,
				))
			}
		}
	}
	return nil
}

// Generate TerraformResources from AWS API,
// from each database create 1 TerraformResource.
// Need only database name as ID for terraform resource
// AWS api support paging
func (g *RedshiftGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := redshift.NewFromConfig(config)

	if err := g.loadClusters(svc); err != nil {
		return err
	}
	if err := g.loadParameterGroups(svc); err != nil {
		return err
	}
	if err := g.loadSubnetGroups(svc); err != nil {
		return err
	}
	if err := g.loadEventSubscription(svc); err != nil {
		return err
	}
	if err := g.loadSnapshotSchedules(svc); err != nil {
		return err
	}

	return nil
}

func (g *RedshiftGenerator) PostConvertHook() error {
	for i, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_redshift_cluster" {
			continue
		}
		for _, parameterGroup := range g.Resources {
			log.Print(parameterGroup.InstanceInfo.Type)
			if parameterGroup.InstanceInfo.Type != "aws_redshift_parameter_group" {
				continue
			}
			if parameterGroup.InstanceState.Attributes["name"] == r.InstanceState.Attributes["cluster_parameter_group_name"] {
				g.Resources[i].Item["cluster_parameter_group_name"] = "${aws_redshift_parameter_group." + parameterGroup.ResourceName + ".name}"
			}
		}

		for _, subnet := range g.Resources {
			if subnet.InstanceInfo.Type != "aws_redshift_subnet_group" {
				continue
			}
			if subnet.InstanceState.Attributes["name"] == r.InstanceState.Attributes["cluster_subnet_group_name"] {
				g.Resources[i].Item["cluster_subnet_group_name"] = "${aws_redshift_subnet_group." + subnet.ResourceName + ".name}"
			}
		}
	}
	return nil
}
