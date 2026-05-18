// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/neptune"
	neptunetypes "github.com/aws/aws-sdk-go-v2/service/neptune/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	neptuneClusterResourceType               = "aws_neptune_cluster"
	neptuneClusterEndpointResourceType       = "aws_neptune_cluster_endpoint"
	neptuneClusterInstanceResourceType       = "aws_neptune_cluster_instance"
	neptuneClusterParameterGroupResourceType = "aws_neptune_cluster_parameter_group"
	neptuneEventSubscriptionResourceType     = "aws_neptune_event_subscription"
	neptuneGlobalClusterResourceType         = "aws_neptune_global_cluster"
	neptuneParameterGroupResourceType        = "aws_neptune_parameter_group"
	neptuneSubnetGroupResourceType           = "aws_neptune_subnet_group"
	neptuneClusterEndpointIDSeparator        = ":"
)

var neptuneAllowEmptyValues = []string{"tags."}

type NeptuneGenerator struct {
	AWSService
}

type neptuneResourceLoader struct {
	serviceName string
	load        func() error
}

func (g *NeptuneGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := neptune.NewFromConfig(config)

	loaders := []neptuneResourceLoader{
		{serviceName: "neptune_cluster", load: func() error { return g.loadClusters(svc) }},
		{serviceName: "neptune_cluster_instance", load: func() error { return g.loadClusterInstances(svc) }},
		{serviceName: "neptune_cluster_endpoint", load: func() error { return g.loadClusterEndpoints(svc) }},
		{serviceName: "neptune_cluster_parameter_group", load: func() error { return g.loadClusterParameterGroups(svc) }},
		{serviceName: "neptune_parameter_group", load: func() error { return g.loadParameterGroups(svc) }},
		{serviceName: "neptune_subnet_group", load: func() error { return g.loadSubnetGroups(svc) }},
		{serviceName: "neptune_event_subscription", load: func() error { return g.loadEventSubscriptions(svc) }},
		{serviceName: "neptune_global_cluster", load: func() error { return g.loadGlobalClusters(svc) }},
	}
	for _, loader := range loaders {
		if !g.shouldLoadNeptuneResource(loader.serviceName) {
			continue
		}
		if err := loader.load(); err != nil {
			return err
		}
	}
	return nil
}

func (g *NeptuneGenerator) loadClusters(svc neptune.DescribeDBClustersAPIClient) error {
	p := neptune.NewDescribeDBClustersPaginator(svc, &neptune.DescribeDBClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.DBClusters {
			if resource, ok := newNeptuneClusterResource(cluster); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NeptuneGenerator) loadClusterInstances(svc neptune.DescribeDBInstancesAPIClient) error {
	p := neptune.NewDescribeDBInstancesPaginator(svc, &neptune.DescribeDBInstancesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, instance := range page.DBInstances {
			if resource, ok := newNeptuneClusterInstanceResource(instance); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NeptuneGenerator) loadClusterEndpoints(svc neptune.DescribeDBClusterEndpointsAPIClient) error {
	p := neptune.NewDescribeDBClusterEndpointsPaginator(svc, &neptune.DescribeDBClusterEndpointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpoint := range page.DBClusterEndpoints {
			if resource, ok := newNeptuneClusterEndpointResource(endpoint); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NeptuneGenerator) loadClusterParameterGroups(svc neptune.DescribeDBClusterParameterGroupsAPIClient) error {
	p := neptune.NewDescribeDBClusterParameterGroupsPaginator(svc, &neptune.DescribeDBClusterParameterGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, parameterGroup := range page.DBClusterParameterGroups {
			if resource, ok := newNeptuneClusterParameterGroupResource(parameterGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NeptuneGenerator) loadParameterGroups(svc neptune.DescribeDBParameterGroupsAPIClient) error {
	p := neptune.NewDescribeDBParameterGroupsPaginator(svc, &neptune.DescribeDBParameterGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, parameterGroup := range page.DBParameterGroups {
			if resource, ok := newNeptuneParameterGroupResource(parameterGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NeptuneGenerator) loadSubnetGroups(svc neptune.DescribeDBSubnetGroupsAPIClient) error {
	p := neptune.NewDescribeDBSubnetGroupsPaginator(svc, &neptune.DescribeDBSubnetGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, subnetGroup := range page.DBSubnetGroups {
			if resource, ok := newNeptuneSubnetGroupResource(subnetGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NeptuneGenerator) loadEventSubscriptions(svc neptune.DescribeEventSubscriptionsAPIClient) error {
	p := neptune.NewDescribeEventSubscriptionsPaginator(svc, &neptune.DescribeEventSubscriptionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, subscription := range page.EventSubscriptionsList {
			if resource, ok := newNeptuneEventSubscriptionResource(subscription); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NeptuneGenerator) loadGlobalClusters(svc neptune.DescribeGlobalClustersAPIClient) error {
	p := neptune.NewDescribeGlobalClustersPaginator(svc, &neptune.DescribeGlobalClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.GlobalClusters {
			if resource, ok := newNeptuneGlobalClusterResource(cluster); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newNeptuneClusterResource(cluster neptunetypes.DBCluster) (terraformutils.Resource, bool) {
	clusterID := StringValue(cluster.DBClusterIdentifier)
	if clusterID == "" || !neptuneRuntimeStatusImportable(StringValue(cluster.Status)) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		clusterID,
		neptuneResourceName("cluster", clusterID),
		neptuneClusterResourceType,
		"aws",
		map[string]string{"cluster_identifier": clusterID},
		neptuneAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNeptuneClusterInstanceResource(instance neptunetypes.DBInstance) (terraformutils.Resource, bool) {
	instanceID := StringValue(instance.DBInstanceIdentifier)
	clusterID := StringValue(instance.DBClusterIdentifier)
	if instanceID == "" || clusterID == "" || !neptuneRuntimeStatusImportable(StringValue(instance.DBInstanceStatus)) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		instanceID,
		neptuneResourceName("cluster_instance", clusterID, instanceID),
		neptuneClusterInstanceResourceType,
		"aws",
		map[string]string{
			"cluster_identifier": clusterID,
			"identifier":         instanceID,
		},
		neptuneAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNeptuneClusterEndpointResource(endpoint neptunetypes.DBClusterEndpoint) (terraformutils.Resource, bool) {
	clusterID := StringValue(endpoint.DBClusterIdentifier)
	endpointID := StringValue(endpoint.DBClusterEndpointIdentifier)
	if clusterID == "" || endpointID == "" || !neptuneCustomClusterEndpoint(endpoint) || !neptuneEndpointStatusImportable(StringValue(endpoint.Status)) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		neptuneClusterEndpointImportID(clusterID, endpointID),
		neptuneResourceName("cluster_endpoint", clusterID, endpointID),
		neptuneClusterEndpointResourceType,
		"aws",
		map[string]string{
			"cluster_endpoint_identifier": endpointID,
			"cluster_identifier":          clusterID,
		},
		neptuneAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNeptuneClusterParameterGroupResource(parameterGroup neptunetypes.DBClusterParameterGroup) (terraformutils.Resource, bool) {
	name := StringValue(parameterGroup.DBClusterParameterGroupName)
	family := StringValue(parameterGroup.DBParameterGroupFamily)
	if name == "" || family == "" || neptuneDefaultParameterGroup(name) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		name,
		neptuneResourceName("cluster_parameter_group", name),
		neptuneClusterParameterGroupResourceType,
		"aws",
		map[string]string{
			"family": family,
			"name":   name,
		},
		neptuneAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNeptuneParameterGroupResource(parameterGroup neptunetypes.DBParameterGroup) (terraformutils.Resource, bool) {
	name := StringValue(parameterGroup.DBParameterGroupName)
	family := StringValue(parameterGroup.DBParameterGroupFamily)
	if name == "" || family == "" || neptuneDefaultParameterGroup(name) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		name,
		neptuneResourceName("parameter_group", name),
		neptuneParameterGroupResourceType,
		"aws",
		map[string]string{
			"family": family,
			"name":   name,
		},
		neptuneAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNeptuneSubnetGroupResource(subnetGroup neptunetypes.DBSubnetGroup) (terraformutils.Resource, bool) {
	name := StringValue(subnetGroup.DBSubnetGroupName)
	if name == "" || neptuneDefaultSubnetGroup(name) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		name,
		neptuneResourceName("subnet_group", name),
		neptuneSubnetGroupResourceType,
		"aws",
		map[string]string{"name": name},
		neptuneAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNeptuneEventSubscriptionResource(subscription neptunetypes.EventSubscription) (terraformutils.Resource, bool) {
	name := StringValue(subscription.CustSubscriptionId)
	if name == "" || !neptuneEventSubscriptionStatusImportable(StringValue(subscription.Status)) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		name,
		neptuneResourceName("event_subscription", name),
		neptuneEventSubscriptionResourceType,
		"aws",
		neptuneAllowEmptyValues,
	), true
}

func newNeptuneGlobalClusterResource(cluster neptunetypes.GlobalCluster) (terraformutils.Resource, bool) {
	name := StringValue(cluster.GlobalClusterIdentifier)
	if name == "" || !neptuneGlobalClusterStatusImportable(StringValue(cluster.Status)) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		name,
		neptuneResourceName("global_cluster", name),
		neptuneGlobalClusterResourceType,
		"aws",
		map[string]string{"global_cluster_identifier": name},
		neptuneAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func (g *NeptuneGenerator) shouldLoadNeptuneResource(serviceNames ...string) bool {
	return shouldLoadAWSResourceForTypedFilters(g.Filter, serviceNames...)
}

func neptuneClusterEndpointImportID(clusterID, endpointID string) string {
	return strings.Join([]string{clusterID, endpointID}, neptuneClusterEndpointIDSeparator)
}

func neptuneRuntimeStatusImportable(status string) bool {
	return strings.EqualFold(status, "available") || strings.EqualFold(status, "stopped")
}

func neptuneEndpointStatusImportable(status string) bool {
	return strings.EqualFold(status, "available")
}

func neptuneEventSubscriptionStatusImportable(status string) bool {
	return strings.EqualFold(status, "active")
}

func neptuneGlobalClusterStatusImportable(status string) bool {
	return strings.EqualFold(status, "available")
}

func neptuneCustomClusterEndpoint(endpoint neptunetypes.DBClusterEndpoint) bool {
	return strings.EqualFold(StringValue(endpoint.EndpointType), "CUSTOM")
}

func neptuneDefaultParameterGroup(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), "default.")
}

func neptuneDefaultSubnetGroup(name string) bool {
	return strings.EqualFold(name, "default")
}

func neptuneResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return "neptune_resource"
	}
	return strings.Join(cleanParts, "_")
}

func (g *NeptuneGenerator) PostConvertHook() error {
	return nil
}
