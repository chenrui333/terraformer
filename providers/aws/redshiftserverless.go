// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/redshiftserverless"
	redshiftserverlesstypes "github.com/aws/aws-sdk-go-v2/service/redshiftserverless/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	redshiftServerlessNamespaceResourceType               = "aws_redshiftserverless_namespace"
	redshiftServerlessWorkgroupResourceType               = "aws_redshiftserverless_workgroup"
	redshiftServerlessSnapshotResourceType                = "aws_redshiftserverless_snapshot"
	redshiftServerlessUsageLimitResourceType              = "aws_redshiftserverless_usage_limit"
	redshiftServerlessEndpointAccessResourceType          = "aws_redshiftserverless_endpoint_access"
	redshiftServerlessCustomDomainAssociationResourceType = "aws_redshiftserverless_custom_domain_association"
	redshiftServerlessResourcePolicyResourceType          = "aws_redshiftserverless_resource_policy"

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

type redshiftServerlessOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *RedshiftServerlessGenerator) loadOptionalResources(loaders []redshiftServerlessOptionalResourceLoader) error {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if redshiftServerlessOptionalResourceErrorSkippable(err) {
				log.Printf("Skipping Redshift Serverless %s: %v", loader.name, err)
				continue
			}
			log.Printf("Failed Redshift Serverless %s discovery: %v", loader.name, err)
			return fmt.Errorf("loading Redshift Serverless %s: %w", loader.name, err)
		}
	}
	return nil
}

func redshiftServerlessOptionalResourceErrorSkippable(err error) bool {
	var notFound *redshiftserverlesstypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return true
	}
	var accessDenied *redshiftserverlesstypes.AccessDeniedException
	if errors.As(err, &accessDenied) {
		return true
	}
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && strings.Contains(strings.ToLower(apiErr.ErrorCode()), "accessdenied")
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
	if err := g.loadOptionalResources([]redshiftServerlessOptionalResourceLoader{
		{name: "usage limits", load: func() error { return g.loadUsageLimits(svc) }},
		{name: "endpoint access", load: func() error { return g.loadEndpointAccess(svc) }},
		{name: "custom domain associations", load: func() error { return g.loadCustomDomainAssociations(svc) }},
		{name: "snapshots and resource policies", load: func() error { return g.loadSnapshots(svc) }},
	}); err != nil {
		return err
	}

	return nil
}

func (g *RedshiftServerlessGenerator) PostConvertHook() error {
	for i := range g.Resources {
		if g.Resources[i].InstanceInfo == nil || g.Resources[i].InstanceInfo.Type != redshiftServerlessResourcePolicyResourceType {
			continue
		}
		wrapRedshiftServerlessPolicyHeredoc(g, &g.Resources[i])
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

func (g *RedshiftServerlessGenerator) loadUsageLimits(svc *redshiftserverless.Client) error {
	p := redshiftserverless.NewListUsageLimitsPaginator(svc, &redshiftserverless.ListUsageLimitsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, usageLimit := range page.UsageLimits {
			if resource, ok := newRedshiftServerlessUsageLimitResource(usageLimit); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *RedshiftServerlessGenerator) loadEndpointAccess(svc *redshiftserverless.Client) error {
	p := redshiftserverless.NewListEndpointAccessPaginator(svc, &redshiftserverless.ListEndpointAccessInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpoint := range page.Endpoints {
			if resource, ok := newRedshiftServerlessEndpointAccessResource(endpoint); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *RedshiftServerlessGenerator) loadCustomDomainAssociations(svc *redshiftserverless.Client) error {
	p := redshiftserverless.NewListCustomDomainAssociationsPaginator(svc, &redshiftserverless.ListCustomDomainAssociationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, association := range page.Associations {
			if resource, ok := newRedshiftServerlessCustomDomainAssociationResource(association); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *RedshiftServerlessGenerator) loadSnapshots(svc *redshiftserverless.Client) error {
	p := redshiftserverless.NewListSnapshotsPaginator(svc, &redshiftserverless.ListSnapshotsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, snapshot := range page.Snapshots {
			if resource, ok := newRedshiftServerlessSnapshotResource(snapshot); ok {
				g.Resources = append(g.Resources, resource)
				if err := g.addRedshiftServerlessResourcePolicy(svc, snapshot); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *RedshiftServerlessGenerator) addRedshiftServerlessResourcePolicy(svc *redshiftserverless.Client, snapshot redshiftserverlesstypes.Snapshot) error {
	resourceArn := StringValue(snapshot.SnapshotArn)
	if resourceArn == "" {
		return nil
	}
	output, err := svc.GetResourcePolicy(context.TODO(), &redshiftserverless.GetResourcePolicyInput{ResourceArn: &resourceArn})
	if err != nil {
		if redshiftServerlessOptionalResourceErrorSkippable(err) {
			return nil
		}
		return err
	}
	if output == nil || output.ResourcePolicy == nil {
		return nil
	}
	if resource, ok := newRedshiftServerlessResourcePolicyResource(*output.ResourcePolicy); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func newRedshiftServerlessNamespaceResource(namespace redshiftserverlesstypes.Namespace) (terraformutils.Resource, bool) {
	importID := redshiftServerlessNamespaceImportID(namespace)
	if importID == "" || !redshiftServerlessNamespaceImportable(namespace) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"namespace_name": importID,
	}
	if StringValue(namespace.AdminPasswordSecretArn) != "" {
		attributes["manage_admin_password"] = "true"
	}
	return terraformutils.NewResource(
		importID,
		redshiftServerlessResourceName("namespace", importID),
		redshiftServerlessNamespaceResourceType,
		"aws",
		attributes,
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

func newRedshiftServerlessSnapshotResource(snapshot redshiftserverlesstypes.Snapshot) (terraformutils.Resource, bool) {
	importID := redshiftServerlessSnapshotImportID(snapshot)
	if importID == "" || !redshiftServerlessSnapshotImportable(snapshot) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"namespace_name": StringValue(snapshot.NamespaceName),
		"snapshot_name":  importID,
	}
	if snapshot.SnapshotRetentionPeriod != nil {
		attributes["retention_period"] = strconv.Itoa(int(*snapshot.SnapshotRetentionPeriod))
	}
	return terraformutils.NewResource(
		importID,
		redshiftServerlessResourceName("snapshot", StringValue(snapshot.NamespaceName), importID),
		redshiftServerlessSnapshotResourceType,
		"aws",
		attributes,
		redshiftServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRedshiftServerlessUsageLimitResource(usageLimit redshiftserverlesstypes.UsageLimit) (terraformutils.Resource, bool) {
	importID := redshiftServerlessUsageLimitImportID(usageLimit)
	if importID == "" || !redshiftServerlessUsageLimitImportable(usageLimit) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"amount":       strconv.FormatInt(*usageLimit.Amount, 10),
		"resource_arn": StringValue(usageLimit.ResourceArn),
		"usage_type":   string(usageLimit.UsageType),
	}
	if usageLimit.BreachAction != "" {
		attributes["breach_action"] = string(usageLimit.BreachAction)
	}
	if usageLimit.Period != "" {
		attributes["period"] = string(usageLimit.Period)
	}
	return terraformutils.NewResource(
		importID,
		redshiftServerlessResourceName("usage-limit", StringValue(usageLimit.ResourceArn), importID),
		redshiftServerlessUsageLimitResourceType,
		"aws",
		attributes,
		redshiftServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRedshiftServerlessEndpointAccessResource(endpoint redshiftserverlesstypes.EndpointAccess) (terraformutils.Resource, bool) {
	importID := redshiftServerlessEndpointAccessImportID(endpoint)
	if importID == "" || !redshiftServerlessEndpointAccessImportable(endpoint) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"endpoint_name":  importID,
		"workgroup_name": StringValue(endpoint.WorkgroupName),
	}
	for key, value := range redshiftServerlessStringSliceAttributes("subnet_ids", endpoint.SubnetIds) {
		attributes[key] = value
	}
	securityGroupIDs := redshiftServerlessEndpointAccessSecurityGroupIDs(endpoint.VpcSecurityGroups)
	if len(securityGroupIDs) > 0 {
		for key, value := range redshiftServerlessStringSliceAttributes("vpc_security_group_ids", securityGroupIDs) {
			attributes[key] = value
		}
	}
	return terraformutils.NewResource(
		importID,
		redshiftServerlessResourceName("endpoint-access", StringValue(endpoint.WorkgroupName), importID),
		redshiftServerlessEndpointAccessResourceType,
		"aws",
		attributes,
		redshiftServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRedshiftServerlessCustomDomainAssociationResource(association redshiftserverlesstypes.Association) (terraformutils.Resource, bool) {
	importID := redshiftServerlessCustomDomainAssociationImportID(association)
	if importID == "" || !redshiftServerlessCustomDomainAssociationImportable(association) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"custom_domain_certificate_arn": StringValue(association.CustomDomainCertificateArn),
		"custom_domain_name":            StringValue(association.CustomDomainName),
		"workgroup_name":                StringValue(association.WorkgroupName),
	}
	return terraformutils.NewResource(
		importID,
		redshiftServerlessResourceName("custom-domain-association", StringValue(association.WorkgroupName), StringValue(association.CustomDomainName)),
		redshiftServerlessCustomDomainAssociationResourceType,
		"aws",
		attributes,
		redshiftServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newRedshiftServerlessResourcePolicyResource(resourcePolicy redshiftserverlesstypes.ResourcePolicy) (terraformutils.Resource, bool) {
	importID := redshiftServerlessResourcePolicyImportID(resourcePolicy)
	if importID == "" || !redshiftServerlessResourcePolicyImportable(resourcePolicy) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		redshiftServerlessResourceName("resource-policy", importID),
		redshiftServerlessResourcePolicyResourceType,
		"aws",
		map[string]string{
			"policy":       StringValue(resourcePolicy.Policy),
			"resource_arn": importID,
		},
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

func redshiftServerlessSnapshotImportID(snapshot redshiftserverlesstypes.Snapshot) string {
	return StringValue(snapshot.SnapshotName)
}

func redshiftServerlessUsageLimitImportID(usageLimit redshiftserverlesstypes.UsageLimit) string {
	return StringValue(usageLimit.UsageLimitId)
}

func redshiftServerlessEndpointAccessImportID(endpoint redshiftserverlesstypes.EndpointAccess) string {
	return StringValue(endpoint.EndpointName)
}

func redshiftServerlessCustomDomainAssociationImportID(association redshiftserverlesstypes.Association) string {
	return redshiftServerlessCustomDomainAssociationImportIDFromParts(StringValue(association.WorkgroupName), StringValue(association.CustomDomainName))
}

func redshiftServerlessCustomDomainAssociationImportIDFromParts(workgroupName, customDomainName string) string {
	if workgroupName == "" || customDomainName == "" {
		return ""
	}
	return workgroupName + "," + customDomainName
}

func redshiftServerlessResourcePolicyImportID(resourcePolicy redshiftserverlesstypes.ResourcePolicy) string {
	return StringValue(resourcePolicy.ResourceArn)
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
		redshiftServerlessNamespaceStatusImportable(namespace.Status)
}

func redshiftServerlessWorkgroupImportable(workgroup redshiftserverlesstypes.Workgroup) bool {
	return redshiftServerlessWorkgroupImportID(workgroup) != "" &&
		StringValue(workgroup.NamespaceName) != "" &&
		redshiftServerlessWorkgroupStatusImportable(workgroup.Status)
}

func redshiftServerlessSnapshotImportable(snapshot redshiftserverlesstypes.Snapshot) bool {
	return redshiftServerlessSnapshotImportID(snapshot) != "" &&
		StringValue(snapshot.NamespaceName) != "" &&
		redshiftServerlessSnapshotStatusImportable(snapshot.Status)
}

func redshiftServerlessUsageLimitImportable(usageLimit redshiftserverlesstypes.UsageLimit) bool {
	return redshiftServerlessUsageLimitImportID(usageLimit) != "" &&
		StringValue(usageLimit.ResourceArn) != "" &&
		usageLimit.Amount != nil &&
		*usageLimit.Amount > 0 &&
		usageLimit.UsageType != ""
}

func redshiftServerlessEndpointAccessImportable(endpoint redshiftserverlesstypes.EndpointAccess) bool {
	return redshiftServerlessEndpointAccessImportID(endpoint) != "" &&
		StringValue(endpoint.WorkgroupName) != "" &&
		len(endpoint.SubnetIds) > 0 &&
		redshiftServerlessEndpointAccessStatusImportable(StringValue(endpoint.EndpointStatus))
}

func redshiftServerlessCustomDomainAssociationImportable(association redshiftserverlesstypes.Association) bool {
	return redshiftServerlessCustomDomainAssociationImportID(association) != "" &&
		StringValue(association.CustomDomainCertificateArn) != "" &&
		association.CustomDomainCertificateExpiryTime != nil
}

func redshiftServerlessResourcePolicyImportable(resourcePolicy redshiftserverlesstypes.ResourcePolicy) bool {
	return redshiftServerlessResourcePolicyImportID(resourcePolicy) != "" &&
		StringValue(resourcePolicy.Policy) != ""
}

func redshiftServerlessNamespaceStatusImportable(status redshiftserverlesstypes.NamespaceStatus) bool {
	return status != "" && status != redshiftserverlesstypes.NamespaceStatusDeleting
}

func redshiftServerlessWorkgroupStatusImportable(status redshiftserverlesstypes.WorkgroupStatus) bool {
	return status != "" && status != redshiftserverlesstypes.WorkgroupStatusDeleting
}

func redshiftServerlessSnapshotStatusImportable(status redshiftserverlesstypes.SnapshotStatus) bool {
	return status == redshiftserverlesstypes.SnapshotStatusAvailable
}

func redshiftServerlessEndpointAccessStatusImportable(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "creating", "deleting", "failed":
		return false
	default:
		return true
	}
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

func redshiftServerlessEndpointAccessSecurityGroupIDs(groups []redshiftserverlesstypes.VpcSecurityGroupMembership) []string {
	var ids []string
	for _, group := range groups {
		if id := StringValue(group.VpcSecurityGroupId); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
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

func wrapRedshiftServerlessPolicyHeredoc(g *RedshiftServerlessGenerator, resource *terraformutils.Resource) {
	if resource.Item == nil {
		return
	}
	policy, ok := resource.Item["policy"].(string)
	if !ok || policy == "" {
		return
	}
	resource.Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
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
