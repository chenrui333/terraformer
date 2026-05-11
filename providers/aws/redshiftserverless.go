// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/redshiftserverless"
	redshiftserverlesstypes "github.com/aws/aws-sdk-go-v2/service/redshiftserverless/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	redshiftServerlessNamespaceResourceType = "aws_redshiftserverless_namespace"
	redshiftServerlessWorkgroupResourceType = "aws_redshiftserverless_workgroup"

	redshiftServerlessResourceNameFallback = "redshiftserverless-resource"
)

var redshiftServerlessAllowEmptyValues = []string{
	"tags.",
	"^enhanced_vpc_routing$",
	`^price_performance_target\.\d+\.enabled$`,
	"^publicly_accessible$",
}

type RedshiftServerlessGenerator struct {
	AWSService
}

func (g *RedshiftServerlessGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := redshiftserverless.NewFromConfig(config)

	if err := g.loadNamespaces(svc); err != nil {
		return err
	}
	if err := g.loadWorkgroups(svc); err != nil {
		return err
	}

	return nil
}

func (g *RedshiftServerlessGenerator) loadNamespaces(svc *redshiftserverless.Client) error {
	p := redshiftserverless.NewListNamespacesPaginator(svc, &redshiftserverless.ListNamespacesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, namespace := range page.Namespaces {
			if resource, ok := newRedshiftServerlessNamespaceResource(namespace); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *RedshiftServerlessGenerator) loadWorkgroups(svc *redshiftserverless.Client) error {
	p := redshiftserverless.NewListWorkgroupsPaginator(svc, &redshiftserverless.ListWorkgroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, workgroup := range page.Workgroups {
			if resource, ok := newRedshiftServerlessWorkgroupResource(workgroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newRedshiftServerlessNamespaceResource(namespace redshiftserverlesstypes.Namespace) (terraformutils.Resource, bool) {
	importID := redshiftServerlessNamespaceImportID(namespace)
	if importID == "" || !redshiftServerlessNamespaceImportable(namespace) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		redshiftServerlessResourceName("namespace", importID),
		redshiftServerlessNamespaceResourceType,
		"aws",
		map[string]string{
			"namespace_name": importID,
		},
		redshiftServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRedshiftServerlessWorkgroupResource(workgroup redshiftserverlesstypes.Workgroup) (terraformutils.Resource, bool) {
	importID := redshiftServerlessWorkgroupImportID(workgroup)
	if importID == "" || !redshiftServerlessWorkgroupImportable(workgroup) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		redshiftServerlessResourceName("workgroup", StringValue(workgroup.NamespaceName), importID),
		redshiftServerlessWorkgroupResourceType,
		"aws",
		redshiftServerlessWorkgroupAttributes(workgroup),
		redshiftServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func redshiftServerlessNamespaceImportID(namespace redshiftserverlesstypes.Namespace) string {
	return StringValue(namespace.NamespaceName)
}

func redshiftServerlessWorkgroupImportID(workgroup redshiftserverlesstypes.Workgroup) string {
	return StringValue(workgroup.WorkgroupName)
}

func redshiftServerlessResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return redshiftServerlessResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func redshiftServerlessNamespaceImportable(namespace redshiftserverlesstypes.Namespace) bool {
	return redshiftServerlessNamespaceImportID(namespace) != "" &&
		namespace.Status == redshiftserverlesstypes.NamespaceStatusAvailable
}

func redshiftServerlessWorkgroupImportable(workgroup redshiftserverlesstypes.Workgroup) bool {
	return redshiftServerlessWorkgroupImportID(workgroup) != "" &&
		StringValue(workgroup.NamespaceName) != "" &&
		workgroup.Status == redshiftserverlesstypes.WorkgroupStatusAvailable
}

func redshiftServerlessWorkgroupAttributes(workgroup redshiftserverlesstypes.Workgroup) map[string]string {
	attributes := map[string]string{
		"namespace_name": StringValue(workgroup.NamespaceName),
		"workgroup_name": redshiftServerlessWorkgroupImportID(workgroup),
	}
	if workgroup.BaseCapacity != nil {
		attributes["base_capacity"] = strconv.Itoa(int(*workgroup.BaseCapacity))
	}
	if workgroup.EnhancedVpcRouting != nil {
		attributes["enhanced_vpc_routing"] = strconv.FormatBool(*workgroup.EnhancedVpcRouting)
	}
	if workgroup.MaxCapacity != nil {
		attributes["max_capacity"] = strconv.Itoa(int(*workgroup.MaxCapacity))
	}
	if workgroup.Port != nil {
		attributes["port"] = strconv.Itoa(int(*workgroup.Port))
	}
	if workgroup.PricePerformanceTarget != nil {
		for key, value := range redshiftServerlessPricePerformanceTargetAttributes("price_performance_target", workgroup.PricePerformanceTarget) {
			attributes[key] = value
		}
	}
	if workgroup.PubliclyAccessible != nil {
		attributes["publicly_accessible"] = strconv.FormatBool(*workgroup.PubliclyAccessible)
	}
	for key, value := range redshiftServerlessStringSliceAttributes("security_group_ids", workgroup.SecurityGroupIds) {
		attributes[key] = value
	}
	for key, value := range redshiftServerlessStringSliceAttributes("subnet_ids", workgroup.SubnetIds) {
		attributes[key] = value
	}
	if value := StringValue(workgroup.TrackName); value != "" {
		attributes["track_name"] = value
	}
	return attributes
}

func redshiftServerlessPricePerformanceTargetAttributes(prefix string, target *redshiftserverlesstypes.PerformanceTarget) map[string]string {
	attributes := map[string]string{
		prefix + ".#":         "1",
		prefix + ".0.enabled": strconv.FormatBool(target.Status == redshiftserverlesstypes.PerformanceTargetStatusEnabled),
	}
	if target.Level != nil {
		attributes[prefix+".0.level"] = strconv.Itoa(int(*target.Level))
	}
	return attributes
}

func redshiftServerlessStringSliceAttributes(prefix string, values []string) map[string]string {
	attributes := map[string]string{
		prefix + ".#": strconv.Itoa(len(values)),
	}
	for i, value := range values {
		attributes[prefix+"."+strconv.Itoa(i)] = value
	}
	return attributes
}
