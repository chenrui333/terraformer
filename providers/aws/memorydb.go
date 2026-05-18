// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/memorydb"
	memorydbtypes "github.com/aws/aws-sdk-go-v2/service/memorydb/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	memoryDBACLResourceType            = "aws_memorydb_acl"
	memoryDBClusterResourceType        = "aws_memorydb_cluster"
	memoryDBParameterGroupResourceType = "aws_memorydb_parameter_group"
	memoryDBSubnetGroupResourceType    = "aws_memorydb_subnet_group"
)

var memoryDBAllowEmptyValues = []string{"tags."}

type MemoryDBGenerator struct {
	AWSService
}

type memoryDBResourceLoader struct {
	serviceName string
	load        func() error
}

func (g *MemoryDBGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := memorydb.NewFromConfig(config)

	loaders := []memoryDBResourceLoader{
		{serviceName: "memorydb_cluster", load: func() error { return g.loadClusters(svc) }},
		{serviceName: "memorydb_acl", load: func() error { return g.loadACLs(svc) }},
		{serviceName: "memorydb_parameter_group", load: func() error { return g.loadParameterGroups(svc) }},
		{serviceName: "memorydb_subnet_group", load: func() error { return g.loadSubnetGroups(svc) }},
	}
	for _, loader := range loaders {
		if !g.shouldLoadMemoryDBResource(loader.serviceName) {
			continue
		}
		if err := loader.load(); err != nil {
			return err
		}
	}
	return nil
}

func (g *MemoryDBGenerator) loadClusters(svc memorydb.DescribeClustersAPIClient) error {
	p := memorydb.NewDescribeClustersPaginator(svc, &memorydb.DescribeClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.Clusters {
			if resource, ok := newMemoryDBClusterResource(cluster); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *MemoryDBGenerator) loadACLs(svc memorydb.DescribeACLsAPIClient) error {
	p := memorydb.NewDescribeACLsPaginator(svc, &memorydb.DescribeACLsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, acl := range page.ACLs {
			if resource, ok := newMemoryDBACLResource(acl); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *MemoryDBGenerator) loadParameterGroups(svc memorydb.DescribeParameterGroupsAPIClient) error {
	p := memorydb.NewDescribeParameterGroupsPaginator(svc, &memorydb.DescribeParameterGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, parameterGroup := range page.ParameterGroups {
			if resource, ok := newMemoryDBParameterGroupResource(parameterGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *MemoryDBGenerator) loadSubnetGroups(svc memorydb.DescribeSubnetGroupsAPIClient) error {
	p := memorydb.NewDescribeSubnetGroupsPaginator(svc, &memorydb.DescribeSubnetGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, subnetGroup := range page.SubnetGroups {
			if resource, ok := newMemoryDBSubnetGroupResource(subnetGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newMemoryDBClusterResource(cluster memorydbtypes.Cluster) (terraformutils.Resource, bool) {
	name := StringValue(cluster.Name)
	if name == "" || !memoryDBClusterStatusImportable(StringValue(cluster.Status)) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{"name": name}
	if aclName := StringValue(cluster.ACLName); aclName != "" {
		attributes["acl_name"] = aclName
	}
	if nodeType := StringValue(cluster.NodeType); nodeType != "" {
		attributes["node_type"] = nodeType
	}
	return terraformutils.NewResource(
		name,
		memoryDBResourceName("cluster", name),
		memoryDBClusterResourceType,
		"aws",
		attributes,
		memoryDBAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newMemoryDBACLResource(acl memorydbtypes.ACL) (terraformutils.Resource, bool) {
	name := StringValue(acl.Name)
	if name == "" || memoryDBDefaultACL(name) || !memoryDBACLStatusImportable(StringValue(acl.Status)) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		name,
		memoryDBResourceName("acl", name),
		memoryDBACLResourceType,
		"aws",
		map[string]string{"name": name},
		memoryDBAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newMemoryDBParameterGroupResource(parameterGroup memorydbtypes.ParameterGroup) (terraformutils.Resource, bool) {
	name := StringValue(parameterGroup.Name)
	family := StringValue(parameterGroup.Family)
	if name == "" || family == "" || memoryDBDefaultParameterGroup(name) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		name,
		memoryDBResourceName("parameter_group", name),
		memoryDBParameterGroupResourceType,
		"aws",
		map[string]string{
			"family": family,
			"name":   name,
		},
		memoryDBAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newMemoryDBSubnetGroupResource(subnetGroup memorydbtypes.SubnetGroup) (terraformutils.Resource, bool) {
	name := StringValue(subnetGroup.Name)
	if name == "" || memoryDBDefaultSubnetGroup(name) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		name,
		memoryDBResourceName("subnet_group", name),
		memoryDBSubnetGroupResourceType,
		"aws",
		map[string]string{"name": name},
		memoryDBAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func (g *MemoryDBGenerator) shouldLoadMemoryDBResource(serviceNames ...string) bool {
	return shouldLoadAWSResourceForTypedFilters(g.Filter, serviceNames...)
}

func memoryDBClusterStatusImportable(status string) bool {
	return strings.EqualFold(status, "available")
}

func memoryDBACLStatusImportable(status string) bool {
	return strings.EqualFold(status, "active")
}

func memoryDBDefaultACL(name string) bool {
	return strings.EqualFold(name, "open-access")
}

func memoryDBDefaultParameterGroup(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), "default.")
}

func memoryDBDefaultSubnetGroup(name string) bool {
	return strings.EqualFold(name, "default")
}

func memoryDBResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return "memorydb_resource"
	}
	return strings.Join(cleanParts, "_")
}

func (g *MemoryDBGenerator) PostConvertHook() error {
	return nil
}
