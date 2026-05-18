// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var RDSAllowEmptyValues = []string{"tags."}

type RDSGenerator struct {
	AWSService
}

type rdsOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *RDSGenerator) loadOptionalResources(loaders []rdsOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("Skipping RDS %s: %v", loader.name, err)
		}
	}
}

func rdsStatusImportable(status string) bool {
	switch strings.ToLower(status) {
	case "deleting", "creating", "failed", "migration-failed":
		return false
	}
	return status != ""
}

func (g *RDSGenerator) loadDBClusters(svc *rds.Client) error {
	p := rds.NewDescribeDBClustersPaginator(svc, &rds.DescribeDBClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.DBClusters {
			resourceName := StringValue(cluster.DBClusterIdentifier)
			if resourceName == "" || !rdsStatusImportable(StringValue(cluster.Status)) {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_rds_cluster",
				"aws",
				RDSAllowEmptyValues,
			))
			g.addRDSClusterRoleAssociations(resourceName, cluster.AssociatedRoles)
			g.addRDSClusterActivityStream(cluster)
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBClusterParameterGroups(svc *rds.Client) error {
	p := rds.NewDescribeDBClusterParameterGroupsPaginator(svc, &rds.DescribeDBClusterParameterGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, parameterGroup := range page.DBClusterParameterGroups {
			resourceName := StringValue(parameterGroup.DBClusterParameterGroupName)
			if resourceName == "" || strings.Contains(resourceName, ".") {
				continue // skip default parameter groups like default.aurora-mysql8.0
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				resourceName,
				resourceName,
				"aws_rds_cluster_parameter_group",
				"aws",
				map[string]string{
					"name": resourceName,
				},
				RDSAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBClusterEndpoints(svc *rds.Client) error {
	p := rds.NewDescribeDBClusterEndpointsPaginator(svc, &rds.DescribeDBClusterEndpointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpoint := range page.DBClusterEndpoints {
			endpointID := StringValue(endpoint.DBClusterEndpointIdentifier)
			clusterID := StringValue(endpoint.DBClusterIdentifier)
			if endpointID == "" || clusterID == "" || !rdsCustomClusterEndpoint(endpoint) {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				endpointID,
				rdsResourceName(clusterID, endpointID),
				"aws_rds_cluster_endpoint",
				"aws",
				map[string]string{
					"cluster_endpoint_identifier": endpointID,
					"cluster_identifier":          clusterID,
				},
				RDSAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBClusterSnapshots(svc *rds.Client) error {
	p := rds.NewDescribeDBClusterSnapshotsPaginator(svc, &rds.DescribeDBClusterSnapshotsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, snapshot := range page.DBClusterSnapshots {
			resourceName := StringValue(snapshot.DBClusterSnapshotIdentifier)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_db_cluster_snapshot",
				"aws",
				RDSAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBProxies(svc *rds.Client) error {
	p := rds.NewDescribeDBProxiesPaginator(svc, &rds.DescribeDBProxiesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, db := range page.DBProxies {
			resourceName := StringValue(db.DBProxyName)
			if resourceName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_db_proxy",
				"aws",
				RDSAllowEmptyValues,
			))
			if err := g.loadDBProxyTargetGroups(svc, resourceName); err != nil {
				log.Printf("Skipping RDS DB proxy target groups for %s: %v", resourceName, err)
			}
			if err := g.loadDBProxyEndpoints(svc, resourceName); err != nil {
				log.Printf("Skipping RDS DB proxy endpoints for %s: %v", resourceName, err)
			}
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBProxyTargetGroups(svc *rds.Client, proxyName string) error {
	p := rds.NewDescribeDBProxyTargetGroupsPaginator(svc, &rds.DescribeDBProxyTargetGroupsInput{
		DBProxyName: aws.String(proxyName),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, targetGroup := range page.TargetGroups {
			targetGroupName := StringValue(targetGroup.TargetGroupName)
			if targetGroupName == "" {
				continue
			}
			if rdsDefaultDBProxyTargetGroup(targetGroup) {
				g.Resources = append(g.Resources, terraformutils.NewResource(
					proxyName,
					rdsResourceName(proxyName, targetGroupName),
					"aws_db_proxy_default_target_group",
					"aws",
					map[string]string{
						"db_proxy_name": proxyName,
					},
					RDSAllowEmptyValues,
					map[string]interface{}{},
				))
			}
			if err := g.loadDBProxyTargets(svc, proxyName, targetGroupName); err != nil {
				log.Printf("Skipping RDS DB proxy targets for %s/%s: %v", proxyName, targetGroupName, err)
			}
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBProxyTargets(svc *rds.Client, proxyName, targetGroupName string) error {
	p := rds.NewDescribeDBProxyTargetsPaginator(svc, &rds.DescribeDBProxyTargetsInput{
		DBProxyName:     aws.String(proxyName),
		TargetGroupName: aws.String(targetGroupName),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, target := range page.Targets {
			rdsResourceID := StringValue(target.RdsResourceId)
			targetType := string(target.Type)
			if rdsResourceID == "" || targetType == "" {
				continue
			}
			attributes := map[string]string{
				"db_proxy_name":     proxyName,
				"target_group_name": targetGroupName,
			}
			if target.Type == rdstypes.TargetTypeRdsInstance {
				attributes["db_instance_identifier"] = rdsResourceID
			} else {
				attributes["db_cluster_identifier"] = rdsResourceID
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				rdsDBProxyTargetImportID(proxyName, targetGroupName, targetType, rdsResourceID),
				rdsResourceName(proxyName, targetGroupName, targetType, rdsResourceID),
				"aws_db_proxy_target",
				"aws",
				attributes,
				RDSAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBProxyEndpoints(svc *rds.Client, proxyName string) error {
	p := rds.NewDescribeDBProxyEndpointsPaginator(svc, &rds.DescribeDBProxyEndpointsInput{
		DBProxyName: aws.String(proxyName),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpoint := range page.DBProxyEndpoints {
			endpointName := StringValue(endpoint.DBProxyEndpointName)
			if endpointName == "" || rdsDefaultDBProxyEndpoint(endpoint) {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				rdsDBProxyEndpointImportID(proxyName, endpointName),
				rdsResourceName(proxyName, endpointName),
				"aws_db_proxy_endpoint",
				"aws",
				map[string]string{
					"db_proxy_endpoint_name": endpointName,
					"db_proxy_name":          proxyName,
				},
				RDSAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBInstances(svc *rds.Client) error {
	p := rds.NewDescribeDBInstancesPaginator(svc, &rds.DescribeDBInstancesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, db := range page.DBInstances {
			resourceName := StringValue(db.DBInstanceIdentifier)
			if resourceName == "" || !rdsStatusImportable(StringValue(db.DBInstanceStatus)) {
				continue
			}
			if clusterID := StringValue(db.DBClusterIdentifier); clusterID != "" {
				g.Resources = append(g.Resources, newRDSClusterInstanceResource(resourceName, clusterID))
				continue
			}
			r := terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_db_instance",
				"aws",
				RDSAllowEmptyValues,
			)
			r.IgnoreKeys = append(r.IgnoreKeys, "^name$")
			g.Resources = append(g.Resources, r)
			g.addDBInstanceRoleAssociations(resourceName, db.AssociatedRoles)
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBInstanceSnapshots(svc *rds.Client) error {
	p := rds.NewDescribeDBSnapshotsPaginator(svc, &rds.DescribeDBSnapshotsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, snapshot := range page.DBSnapshots {
			resourceName := StringValue(snapshot.DBSnapshotIdentifier)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_db_snapshot",
				"aws",
				RDSAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RDSGenerator) addDBInstanceRoleAssociations(instanceID string, roles []rdstypes.DBInstanceRole) {
	for _, role := range roles {
		roleARN := StringValue(role.RoleArn)
		featureName := StringValue(role.FeatureName)
		if roleARN == "" || featureName == "" || !rdsRoleAssociationStatusImportable(StringValue(role.Status)) {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			rdsRoleAssociationImportID(instanceID, roleARN),
			rdsResourceName(instanceID, rdsIAMRoleResourceName(roleARN)),
			"aws_db_instance_role_association",
			"aws",
			map[string]string{
				"db_instance_identifier": instanceID,
				"feature_name":           featureName,
				"role_arn":               roleARN,
			},
			RDSAllowEmptyValues,
			map[string]interface{}{},
		))
	}
}

func newRDSClusterInstanceResource(instanceID, clusterID string) terraformutils.Resource {
	return terraformutils.NewResource(
		instanceID,
		rdsCompositeResourceName(clusterID, instanceID),
		"aws_rds_cluster_instance",
		"aws",
		map[string]string{
			"cluster_identifier": clusterID,
			"identifier":         instanceID,
		},
		RDSAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *RDSGenerator) addRDSClusterRoleAssociations(clusterID string, roles []rdstypes.DBClusterRole) {
	for _, role := range roles {
		roleARN := StringValue(role.RoleArn)
		featureName := StringValue(role.FeatureName)
		if clusterID == "" || roleARN == "" || !rdsRoleAssociationStatusImportable(StringValue(role.Status)) {
			continue
		}
		attributes := map[string]string{
			"db_cluster_identifier": clusterID,
			"role_arn":              roleARN,
		}
		if featureName != "" {
			attributes["feature_name"] = featureName
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			rdsRoleAssociationImportID(clusterID, roleARN),
			rdsCompositeResourceName(clusterID, rdsIAMRoleResourceName(roleARN)),
			"aws_rds_cluster_role_association",
			"aws",
			attributes,
			RDSAllowEmptyValues,
			map[string]interface{}{},
		))
	}
}

func (g *RDSGenerator) addRDSClusterActivityStream(cluster rdstypes.DBCluster) {
	clusterARN := StringValue(cluster.DBClusterArn)
	clusterID := StringValue(cluster.DBClusterIdentifier)
	mode := string(cluster.ActivityStreamMode)
	if clusterARN == "" || mode == "" || !rdsActivityStreamStatusImportable(cluster.ActivityStreamStatus) {
		return
	}
	attributes := map[string]string{
		"mode":         mode,
		"resource_arn": clusterARN,
	}
	if kmsKeyID := StringValue(cluster.ActivityStreamKmsKeyId); kmsKeyID != "" {
		attributes["kms_key_id"] = kmsKeyID
	}
	g.Resources = append(g.Resources, terraformutils.NewResource(
		clusterARN,
		rdsCompositeResourceName(clusterID, "activity_stream"),
		"aws_rds_cluster_activity_stream",
		"aws",
		attributes,
		RDSAllowEmptyValues,
		map[string]interface{}{},
	))
}

func (g *RDSGenerator) loadDBParameterGroups(svc *rds.Client) error {
	p := rds.NewDescribeDBParameterGroupsPaginator(svc, &rds.DescribeDBParameterGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, parameterGroup := range page.DBParameterGroups {
			resourceName := StringValue(parameterGroup.DBParameterGroupName)
			if strings.Contains(resourceName, ".") {
				continue // skip default Default ParameterGroups like default.mysql5.6
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_db_parameter_group",
				"aws",
				RDSAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RDSGenerator) loadDBSubnetGroups(svc *rds.Client) error {
	p := rds.NewDescribeDBSubnetGroupsPaginator(svc, &rds.DescribeDBSubnetGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, subnet := range page.DBSubnetGroups {
			resourceName := StringValue(subnet.DBSubnetGroupName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_db_subnet_group",
				"aws",
				RDSAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RDSGenerator) loadOptionGroups(svc *rds.Client) error {
	p := rds.NewDescribeOptionGroupsPaginator(svc, &rds.DescribeOptionGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, optionGroup := range page.OptionGroupsList {
			resourceName := StringValue(optionGroup.OptionGroupName)
			if strings.Contains(resourceName, ".") || strings.Contains(resourceName, ":") {
				continue // skip default Default OptionGroups like default.mysql5.6
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_db_option_group",
				"aws",
				RDSAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RDSGenerator) loadEventSubscription(svc *rds.Client) error {
	p := rds.NewDescribeEventSubscriptionsPaginator(svc, &rds.DescribeEventSubscriptionsInput{})
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
				"aws_db_event_subscription",
				"aws",
				RDSAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *RDSGenerator) loadRDSGlobalClusters(svc *rds.Client) error {
	p := rds.NewDescribeGlobalClustersPaginator(svc, &rds.DescribeGlobalClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.GlobalClusters {
			resourceName := StringValue(cluster.GlobalClusterIdentifier)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_rds_global_cluster",
				"aws",
				RDSAllowEmptyValues,
			))
		}
	}
	return nil
}

// Generate TerraformResources from AWS API,
// from each database create 1 TerraformResource.
// Need only database name as ID for terraform resource
// AWS api support paging
func (g *RDSGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := rds.NewFromConfig(config)

	if err := g.loadDBClusters(svc); err != nil {
		return err
	}
	if err := g.loadDBClusterSnapshots(svc); err != nil {
		return err
	}
	if err := g.loadDBInstances(svc); err != nil {
		return err
	}
	if err := g.loadDBInstanceSnapshots(svc); err != nil {
		return err
	}
	if err := g.loadDBProxies(svc); err != nil {
		return err
	}
	if err := g.loadDBParameterGroups(svc); err != nil {
		return err
	}
	if err := g.loadDBSubnetGroups(svc); err != nil {
		return err
	}
	if err := g.loadOptionGroups(svc); err != nil {
		return err
	}

	if err := g.loadEventSubscription(svc); err != nil {
		return err
	}

	if err := g.loadRDSGlobalClusters(svc); err != nil {
		return err
	}

	g.loadOptionalResources([]rdsOptionalResourceLoader{
		{name: "cluster parameter groups", load: func() error { return g.loadDBClusterParameterGroups(svc) }},
		{name: "custom cluster endpoints", load: func() error { return g.loadDBClusterEndpoints(svc) }},
	})

	return nil
}

func rdsDBProxyEndpointImportID(proxyName, endpointName string) string {
	return strings.Join([]string{proxyName, endpointName}, "/")
}

func rdsDBProxyTargetImportID(proxyName, targetGroupName, targetType, resourceID string) string {
	return strings.Join([]string{proxyName, targetGroupName, targetType, resourceID}, "/")
}

func rdsRoleAssociationImportID(resourceID, roleARN string) string {
	return fmt.Sprintf("%s,%s", resourceID, roleARN)
}

func rdsRoleAssociationStatusImportable(status string) bool {
	return strings.EqualFold(status, "active")
}

func rdsActivityStreamStatusImportable(status rdstypes.ActivityStreamStatus) bool {
	return strings.EqualFold(string(status), "started")
}

func rdsCompositeResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return "rds_resource"
	}
	encoded := make([]string, 0, len(cleanParts))
	for _, part := range cleanParts {
		encoded = append(encoded, fmt.Sprintf("%d_%s", len(part), part))
	}
	return strings.Join(encoded, "__")
}

func rdsResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return "rds_resource"
	}
	return strings.Join(cleanParts, "_")
}

func rdsIAMRoleResourceName(roleARN string) string {
	resource := arnLastSegment(roleARN, ":")
	return strings.TrimPrefix(resource, "role/")
}

func rdsCustomClusterEndpoint(endpoint rdstypes.DBClusterEndpoint) bool {
	return StringValue(endpoint.EndpointType) == "CUSTOM"
}

func rdsDefaultDBProxyEndpoint(endpoint rdstypes.DBProxyEndpoint) bool {
	return endpoint.IsDefault != nil && *endpoint.IsDefault
}

func rdsDefaultDBProxyTargetGroup(targetGroup rdstypes.DBProxyTargetGroup) bool {
	return targetGroup.IsDefault != nil && *targetGroup.IsDefault
}

func (g *RDSGenerator) PostConvertHook() error {
	for i, r := range g.Resources {
		if r.InstanceInfo.Type == "aws_db_instance" || r.InstanceInfo.Type == "aws_rds_cluster" {
			for _, dbInstance := range g.Resources {
				if dbInstance.InstanceInfo.Type != "aws_db_instance" {
					continue
				}
				if g.Resources[i].Item["replicate_source_db"] != nil {
					delete(g.Resources[i].Item, "username")
					delete(g.Resources[i].Item, "engine_version")
					delete(g.Resources[i].Item, "engine")
					delete(g.Resources[i].Item, "db_name")
				}
			}

			for _, parameterGroup := range g.Resources {
				if parameterGroup.InstanceInfo.Type != "aws_db_parameter_group" {
					continue
				}
				if parameterGroup.InstanceState.Attributes["name"] == r.InstanceState.Attributes["parameter_group_name"] {
					g.Resources[i].Item["parameter_group_name"] = "${aws_db_parameter_group." + parameterGroup.ResourceName + ".name}"
				}
			}

			for _, subnet := range g.Resources {
				if subnet.InstanceInfo.Type != "aws_db_subnet_group" {
					continue
				}
				if subnet.InstanceState.Attributes["name"] == r.InstanceState.Attributes["db_subnet_group_name"] {
					g.Resources[i].Item["db_subnet_group_name"] = "${aws_db_subnet_group." + subnet.ResourceName + ".name}"
				}
			}

			for _, optionGroup := range g.Resources {
				if optionGroup.InstanceInfo.Type != "aws_db_option_group" {
					continue
				}
				if optionGroup.InstanceState.Attributes["name"] == r.InstanceState.Attributes["option_group_name"] {
					g.Resources[i].Item["option_group_name"] = "${aws_db_option_group." + optionGroup.ResourceName + ".name}"
				}
			}
		} else {
			continue
		}
	}
	return nil
}
