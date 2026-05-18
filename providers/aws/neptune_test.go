// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/neptune"
	neptunetypes "github.com/aws/aws-sdk-go-v2/service/neptune/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewNeptuneClusterResource(t *testing.T) {
	resource, ok := newNeptuneClusterResource(neptunetypes.DBCluster{
		DBClusterIdentifier: aws.String("graph-prod"),
		Status:              aws.String("available"),
	})
	if !ok {
		t.Fatal("newNeptuneClusterResource() ok = false, want true")
	}
	if got, want := resource.InstanceInfo.Type, neptuneClusterResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.ResourceName, "tfer--7_cluster__10_graph-prod"; got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["cluster_identifier"], "graph-prod"; got != want {
		t.Fatalf("cluster_identifier = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(resource.IgnoreKeys, "^cluster_identifier_prefix$") {
		t.Fatalf("cluster IgnoreKeys = %v, want ^cluster_identifier_prefix$", resource.IgnoreKeys)
	}

	if _, ok := newNeptuneClusterResource(neptunetypes.DBCluster{
		DBClusterIdentifier: aws.String("graph-prod"),
		Status:              aws.String("creating"),
	}); ok {
		t.Fatal("creating cluster should be skipped")
	}
}

func TestNewNeptuneClusterInstanceResource(t *testing.T) {
	resource, ok := newNeptuneClusterInstanceResource(neptunetypes.DBInstance{
		DBClusterIdentifier:  aws.String("graph-prod"),
		DBInstanceIdentifier: aws.String("graph-prod-1"),
		DBInstanceStatus:     aws.String("available"),
	})
	if !ok {
		t.Fatal("newNeptuneClusterInstanceResource() ok = false, want true")
	}
	if got, want := resource.InstanceInfo.Type, neptuneClusterInstanceResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.ID, "graph-prod-1"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["cluster_identifier"], "graph-prod"; got != want {
		t.Fatalf("cluster_identifier = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(resource.IgnoreKeys, "^identifier_prefix$") {
		t.Fatalf("cluster instance IgnoreKeys = %v, want ^identifier_prefix$", resource.IgnoreKeys)
	}

	if _, ok := newNeptuneClusterInstanceResource(neptunetypes.DBInstance{
		DBInstanceIdentifier: aws.String("graph-prod-1"),
		DBInstanceStatus:     aws.String("available"),
	}); ok {
		t.Fatal("cluster instance with empty cluster ID should be skipped")
	}
}

func TestNewNeptuneClusterEndpointResource(t *testing.T) {
	resource, ok := newNeptuneClusterEndpointResource(neptunetypes.DBClusterEndpoint{
		DBClusterEndpointIdentifier: aws.String("custom-reader"),
		DBClusterIdentifier:         aws.String("graph-prod"),
		EndpointType:                aws.String("CUSTOM"),
		Status:                      aws.String("available"),
	})
	if !ok {
		t.Fatal("newNeptuneClusterEndpointResource() ok = false, want true")
	}
	if got, want := resource.InstanceInfo.Type, neptuneClusterEndpointResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.ID, "graph-prod:custom-reader"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["cluster_endpoint_identifier"], "custom-reader"; got != want {
		t.Fatalf("cluster_endpoint_identifier = %q, want %q", got, want)
	}

	if _, ok := newNeptuneClusterEndpointResource(neptunetypes.DBClusterEndpoint{
		DBClusterEndpointIdentifier: aws.String("reader"),
		DBClusterIdentifier:         aws.String("graph-prod"),
		EndpointType:                aws.String("READER"),
		Status:                      aws.String("available"),
	}); ok {
		t.Fatal("default reader endpoint should be skipped")
	}
	if _, ok := newNeptuneClusterEndpointResource(neptunetypes.DBClusterEndpoint{
		DBClusterEndpointIdentifier: aws.String("custom-reader"),
		DBClusterIdentifier:         aws.String("graph-prod"),
		EndpointType:                aws.String("CUSTOM"),
		Status:                      aws.String("creating"),
	}); ok {
		t.Fatal("creating endpoint should be skipped")
	}
}

func TestNewNeptuneParameterGroupResources(t *testing.T) {
	clusterGroup, ok := newNeptuneClusterParameterGroupResource(neptunetypes.DBClusterParameterGroup{
		DBClusterParameterGroupName: aws.String("graph-cluster-params"),
		DBParameterGroupFamily:      aws.String("neptune1.3"),
	})
	if !ok {
		t.Fatal("newNeptuneClusterParameterGroupResource() ok = false, want true")
	}
	if got, want := clusterGroup.InstanceInfo.Type, neptuneClusterParameterGroupResourceType; got != want {
		t.Fatalf("cluster parameter group resource type = %q, want %q", got, want)
	}
	if got, want := clusterGroup.InstanceState.Attributes["family"], "neptune1.3"; got != want {
		t.Fatalf("cluster parameter group family = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(clusterGroup.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("cluster parameter group IgnoreKeys = %v, want ^name_prefix$", clusterGroup.IgnoreKeys)
	}

	parameterGroup, ok := newNeptuneParameterGroupResource(neptunetypes.DBParameterGroup{
		DBParameterGroupFamily: aws.String("neptune1.3"),
		DBParameterGroupName:   aws.String("graph-params"),
	})
	if !ok {
		t.Fatal("newNeptuneParameterGroupResource() ok = false, want true")
	}
	if got, want := parameterGroup.InstanceInfo.Type, neptuneParameterGroupResourceType; got != want {
		t.Fatalf("parameter group resource type = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(parameterGroup.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("parameter group IgnoreKeys = %v, want ^name_prefix$", parameterGroup.IgnoreKeys)
	}

	if _, ok := newNeptuneClusterParameterGroupResource(neptunetypes.DBClusterParameterGroup{
		DBClusterParameterGroupName: aws.String("default.neptune1.3"),
		DBParameterGroupFamily:      aws.String("neptune1.3"),
	}); ok {
		t.Fatal("default cluster parameter group should be skipped")
	}
	if _, ok := newNeptuneParameterGroupResource(neptunetypes.DBParameterGroup{
		DBParameterGroupName: aws.String("graph-params"),
	}); ok {
		t.Fatal("parameter group with empty family should be skipped")
	}
}

func TestNewNeptuneSubnetAndEventResources(t *testing.T) {
	subnetGroup, ok := newNeptuneSubnetGroupResource(neptunetypes.DBSubnetGroup{
		DBSubnetGroupName: aws.String("graph-subnets"),
	})
	if !ok {
		t.Fatal("newNeptuneSubnetGroupResource() ok = false, want true")
	}
	if got, want := subnetGroup.InstanceInfo.Type, neptuneSubnetGroupResourceType; got != want {
		t.Fatalf("subnet group resource type = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(subnetGroup.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("subnet group IgnoreKeys = %v, want ^name_prefix$", subnetGroup.IgnoreKeys)
	}
	if _, ok := newNeptuneSubnetGroupResource(neptunetypes.DBSubnetGroup{DBSubnetGroupName: aws.String("default")}); ok {
		t.Fatal("default subnet group should be skipped")
	}

	events, ok := newNeptuneEventSubscriptionResource(neptunetypes.EventSubscription{
		CustSubscriptionId: aws.String("graph-events"),
		Status:             aws.String("active"),
	})
	if !ok {
		t.Fatal("newNeptuneEventSubscriptionResource() ok = false, want true")
	}
	if got, want := events.InstanceInfo.Type, neptuneEventSubscriptionResourceType; got != want {
		t.Fatalf("event subscription resource type = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(events.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("event subscription IgnoreKeys = %v, want ^name_prefix$", events.IgnoreKeys)
	}
	if _, ok := newNeptuneEventSubscriptionResource(neptunetypes.EventSubscription{
		CustSubscriptionId: aws.String("graph-events"),
		Status:             aws.String("no-permission"),
	}); ok {
		t.Fatal("no-permission event subscription should be skipped")
	}
}

func TestNeptuneShouldLoadResource(t *testing.T) {
	g := NeptuneGenerator{}
	if !g.shouldLoadNeptuneResource("neptune_cluster") {
		t.Fatal("should load Neptune cluster without typed filters")
	}

	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "neptune_cluster_endpoint",
		FieldPath:        "id",
		AcceptableValues: []string{"graph-prod:custom-reader"},
	}}
	if !g.shouldLoadNeptuneResource("neptune_cluster_endpoint") {
		t.Fatal("should load matching typed Neptune endpoint filter")
	}
	if g.shouldLoadNeptuneResource("neptune_cluster") {
		t.Fatal("should not load Neptune clusters for typed endpoint filter")
	}
}

func TestNeptuneLoadClusterEndpointsPaginates(t *testing.T) {
	g := &NeptuneGenerator{}
	client := &fakeNeptuneDescribeClusterEndpointsClient{
		t: t,
		pages: []*neptune.DescribeDBClusterEndpointsOutput{
			{
				DBClusterEndpoints: []neptunetypes.DBClusterEndpoint{
					{
						DBClusterEndpointIdentifier: aws.String("custom-a"),
						DBClusterIdentifier:         aws.String("graph-prod"),
						EndpointType:                aws.String("CUSTOM"),
						Status:                      aws.String("available"),
					},
				},
				Marker: aws.String("page-2"),
			},
			{
				DBClusterEndpoints: []neptunetypes.DBClusterEndpoint{
					{
						DBClusterEndpointIdentifier: aws.String("custom-b"),
						DBClusterIdentifier:         aws.String("graph-prod"),
						EndpointType:                aws.String("CUSTOM"),
						Status:                      aws.String("available"),
					},
					{
						DBClusterEndpointIdentifier: aws.String("reader"),
						DBClusterIdentifier:         aws.String("graph-prod"),
						EndpointType:                aws.String("READER"),
						Status:                      aws.String("available"),
					},
				},
			},
		},
	}

	if err := g.loadClusterEndpoints(client); err != nil {
		t.Fatalf("loadClusterEndpoints() error = %v", err)
	}
	if got, want := client.calls, 2; got != want {
		t.Fatalf("DescribeDBClusterEndpoints calls = %d, want %d", got, want)
	}
	if got, want := client.markers, []string{"", "page-2"}; !stringSlicesEqual(got, want) {
		t.Fatalf("DescribeDBClusterEndpoints markers = %#v, want %#v", got, want)
	}
	if got, want := len(g.Resources), 2; got != want {
		t.Fatalf("len(Resources) = %d, want %d", got, want)
	}
}

func TestNeptuneStatusPredicatesAndImportID(t *testing.T) {
	if got, want := neptuneClusterEndpointImportID("graph-prod", "custom-reader"), "graph-prod:custom-reader"; got != want {
		t.Fatalf("neptuneClusterEndpointImportID() = %q, want %q", got, want)
	}
	if !neptuneRuntimeStatusImportable("AVAILABLE") {
		t.Fatal("AVAILABLE runtime state should be importable")
	}
	if !neptuneRuntimeStatusImportable("stopped") {
		t.Fatal("stopped runtime state should be importable")
	}
	if neptuneRuntimeStatusImportable("backing-up") {
		t.Fatal("backing-up runtime state should be skipped")
	}
	if !neptuneEndpointStatusImportable("available") {
		t.Fatal("available endpoint should be importable")
	}
	if neptuneEndpointStatusImportable("modifying") {
		t.Fatal("modifying endpoint should be skipped")
	}
}

func TestNeptuneResourceNameIsCollisionResistant(t *testing.T) {
	if got, want := neptuneResourceName(), "neptune_resource"; got != want {
		t.Fatalf("neptuneResourceName() = %q, want %q", got, want)
	}
	if got, want := neptuneResourceName("cluster", "graph-prod"), "7_cluster__10_graph-prod"; got != want {
		t.Fatalf("neptuneResourceName() = %q, want %q", got, want)
	}
	first := neptuneResourceName("ab", "c")
	second := neptuneResourceName("a", "bc")
	if first == second {
		t.Fatalf("neptuneResourceName collision: %q == %q", first, second)
	}
}

type fakeNeptuneDescribeClusterEndpointsClient struct {
	t       *testing.T
	pages   []*neptune.DescribeDBClusterEndpointsOutput
	calls   int
	markers []string
}

func (c *fakeNeptuneDescribeClusterEndpointsClient) DescribeDBClusterEndpoints(_ context.Context, input *neptune.DescribeDBClusterEndpointsInput, _ ...func(*neptune.Options)) (*neptune.DescribeDBClusterEndpointsOutput, error) {
	c.t.Helper()
	if input.Marker == nil {
		c.markers = append(c.markers, "")
	} else {
		c.markers = append(c.markers, *input.Marker)
	}
	if c.calls >= len(c.pages) {
		c.t.Fatalf("unexpected DescribeDBClusterEndpoints call %d", c.calls+1)
	}
	page := c.pages[c.calls]
	c.calls++
	return page, nil
}
