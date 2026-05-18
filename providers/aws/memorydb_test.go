// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/memorydb"
	memorydbtypes "github.com/aws/aws-sdk-go-v2/service/memorydb/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewMemoryDBClusterResource(t *testing.T) {
	resource, ok := newMemoryDBClusterResource(memorydbtypes.Cluster{
		ACLName:  aws.String("app-acl"),
		Name:     aws.String("cache-prod"),
		NodeType: aws.String("db.r7g.large"),
		Status:   aws.String("available"),
	})
	if !ok {
		t.Fatal("newMemoryDBClusterResource() ok = false, want true")
	}
	if got, want := resource.InstanceInfo.Type, memoryDBClusterResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.ResourceName, "tfer--cluster_cache-prod"; got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["acl_name"], "app-acl"; got != want {
		t.Fatalf("acl_name = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["node_type"], "db.r7g.large"; got != want {
		t.Fatalf("node_type = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(resource.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("cluster IgnoreKeys = %v, want ^name_prefix$", resource.IgnoreKeys)
	}

	if _, ok := newMemoryDBClusterResource(memorydbtypes.Cluster{
		Name:   aws.String("cache-prod"),
		Status: aws.String("creating"),
	}); ok {
		t.Fatal("creating cluster should be skipped")
	}
	if _, ok := newMemoryDBClusterResource(memorydbtypes.Cluster{
		Status: aws.String("available"),
	}); ok {
		t.Fatal("cluster with empty name should be skipped")
	}
}

func TestNewMemoryDBACLResource(t *testing.T) {
	resource, ok := newMemoryDBACLResource(memorydbtypes.ACL{
		Name:   aws.String("app-acl"),
		Status: aws.String("active"),
	})
	if !ok {
		t.Fatal("newMemoryDBACLResource() ok = false, want true")
	}
	if got, want := resource.InstanceInfo.Type, memoryDBACLResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.ResourceName, "tfer--acl_app-acl"; got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(resource.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("ACL IgnoreKeys = %v, want ^name_prefix$", resource.IgnoreKeys)
	}

	if _, ok := newMemoryDBACLResource(memorydbtypes.ACL{
		Name:   aws.String("open-access"),
		Status: aws.String("active"),
	}); ok {
		t.Fatal("default open-access ACL should be skipped")
	}
	if _, ok := newMemoryDBACLResource(memorydbtypes.ACL{
		Name:   aws.String("app-acl"),
		Status: aws.String("modifying"),
	}); ok {
		t.Fatal("modifying ACL should be skipped")
	}
}

func TestNewMemoryDBParameterGroupResource(t *testing.T) {
	resource, ok := newMemoryDBParameterGroupResource(memorydbtypes.ParameterGroup{
		Family: aws.String("memorydb_redis7"),
		Name:   aws.String("app-params"),
	})
	if !ok {
		t.Fatal("newMemoryDBParameterGroupResource() ok = false, want true")
	}
	if got, want := resource.InstanceInfo.Type, memoryDBParameterGroupResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["family"], "memorydb_redis7"; got != want {
		t.Fatalf("family = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(resource.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("parameter group IgnoreKeys = %v, want ^name_prefix$", resource.IgnoreKeys)
	}

	if _, ok := newMemoryDBParameterGroupResource(memorydbtypes.ParameterGroup{
		Family: aws.String("memorydb_redis7"),
		Name:   aws.String("default.memorydb-redis7"),
	}); ok {
		t.Fatal("default parameter group should be skipped")
	}
	if _, ok := newMemoryDBParameterGroupResource(memorydbtypes.ParameterGroup{
		Name: aws.String("app-params"),
	}); ok {
		t.Fatal("parameter group with empty family should be skipped")
	}
}

func TestNewMemoryDBSubnetGroupResource(t *testing.T) {
	resource, ok := newMemoryDBSubnetGroupResource(memorydbtypes.SubnetGroup{
		Name: aws.String("app-subnets"),
	})
	if !ok {
		t.Fatal("newMemoryDBSubnetGroupResource() ok = false, want true")
	}
	if got, want := resource.InstanceInfo.Type, memoryDBSubnetGroupResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.ResourceName, "tfer--subnet_group_app-subnets"; got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(resource.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("subnet group IgnoreKeys = %v, want ^name_prefix$", resource.IgnoreKeys)
	}

	if _, ok := newMemoryDBSubnetGroupResource(memorydbtypes.SubnetGroup{
		Name: aws.String("default"),
	}); ok {
		t.Fatal("default subnet group should be skipped")
	}
}

func TestMemoryDBShouldLoadResource(t *testing.T) {
	g := MemoryDBGenerator{}
	if !g.shouldLoadMemoryDBResource("memorydb_cluster") {
		t.Fatal("should load MemoryDB cluster without typed filters")
	}

	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "memorydb_acl",
		FieldPath:        "id",
		AcceptableValues: []string{"app-acl"},
	}}
	if !g.shouldLoadMemoryDBResource("memorydb_acl") {
		t.Fatal("should load matching typed MemoryDB ACL filter")
	}
	if g.shouldLoadMemoryDBResource("memorydb_cluster") {
		t.Fatal("should not load MemoryDB clusters for typed ACL filter")
	}
}

func TestMemoryDBLoadClustersPaginates(t *testing.T) {
	g := &MemoryDBGenerator{}
	client := &fakeMemoryDBDescribeClustersClient{
		t: t,
		pages: []*memorydb.DescribeClustersOutput{
			{
				Clusters: []memorydbtypes.Cluster{
					{Name: aws.String("cache-a"), Status: aws.String("available")},
				},
				NextToken: aws.String("page-2"),
			},
			{
				Clusters: []memorydbtypes.Cluster{
					{Name: aws.String("cache-b"), Status: aws.String("available")},
					{Name: aws.String("cache-c"), Status: aws.String("creating")},
				},
			},
		},
	}

	if err := g.loadClusters(client); err != nil {
		t.Fatalf("loadClusters() error = %v", err)
	}
	if got, want := client.calls, 2; got != want {
		t.Fatalf("DescribeClusters calls = %d, want %d", got, want)
	}
	if got, want := client.tokens, []string{"", "page-2"}; !stringSlicesEqual(got, want) {
		t.Fatalf("DescribeClusters tokens = %#v, want %#v", got, want)
	}
	if got, want := len(g.Resources), 2; got != want {
		t.Fatalf("len(Resources) = %d, want %d", got, want)
	}
}

func TestMemoryDBStatusAndDefaultPredicates(t *testing.T) {
	if !memoryDBClusterStatusImportable("AVAILABLE") {
		t.Fatal("AVAILABLE cluster should be importable")
	}
	if memoryDBClusterStatusImportable("updating") {
		t.Fatal("updating cluster should be skipped")
	}
	if !memoryDBACLStatusImportable("ACTIVE") {
		t.Fatal("ACTIVE ACL should be importable")
	}
	if !memoryDBDefaultACL("open-access") {
		t.Fatal("open-access should be recognized as the default ACL")
	}
	if !memoryDBDefaultParameterGroup("default.memorydb-redis7") {
		t.Fatal("default parameter group should be recognized")
	}
	if !memoryDBDefaultSubnetGroup("default") {
		t.Fatal("default subnet group should be recognized")
	}
}

type fakeMemoryDBDescribeClustersClient struct {
	t      *testing.T
	pages  []*memorydb.DescribeClustersOutput
	calls  int
	tokens []string
}

func (c *fakeMemoryDBDescribeClustersClient) DescribeClusters(_ context.Context, input *memorydb.DescribeClustersInput, _ ...func(*memorydb.Options)) (*memorydb.DescribeClustersOutput, error) {
	c.t.Helper()
	if input.NextToken == nil {
		c.tokens = append(c.tokens, "")
	} else {
		c.tokens = append(c.tokens, *input.NextToken)
	}
	if c.calls >= len(c.pages) {
		c.t.Fatalf("unexpected DescribeClusters call %d", c.calls+1)
	}
	page := c.pages[c.calls]
	c.calls++
	return page, nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
