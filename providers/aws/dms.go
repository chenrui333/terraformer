// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	dmstypes "github.com/aws/aws-sdk-go-v2/service/databasemigrationservice/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	dmsCertificateResourceType            = "aws_dms_certificate"
	dmsEventSubscriptionResourceType      = "aws_dms_event_subscription"
	dmsReplicationConfigResourceType      = "aws_dms_replication_config"
	dmsReplicationInstanceResourceType    = "aws_dms_replication_instance"
	dmsReplicationSubnetGroupResourceType = "aws_dms_replication_subnet_group"
	dmsEndpointResourceType               = "aws_dms_endpoint"
	dmsReplicationTaskResourceType        = "aws_dms_replication_task"

	dmsEventSubscriptionStatusActive = "active"

	dmsEventSubscriptionSourceTypeReplicationInstance = "replication-instance"
	dmsEventSubscriptionSourceTypeReplicationTask     = "replication-task"

	dmsReplicationInstanceStatusAvailable = "available"
	dmsReplicationTaskStatusReady         = "ready"
	dmsReplicationTaskStatusRunning       = "running"
	dmsReplicationTaskStatusStopped       = "stopped"

	dmsEndpointEngineS3 = "s3"
)

var dmsAllowEmptyValues = []string{"tags.", "^enabled$"}

type DmsGenerator struct {
	AWSService
}

type dmsOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *DmsGenerator) loadOptionalResources(loaders []dmsOptionalResourceLoader) error {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if dmsOptionalResourceErrorSkippable(err) {
				log.Printf("Skipping DMS %s: %v", loader.name, err)
				continue
			}
			log.Printf("Failed DMS %s discovery: %v", loader.name, err)
			return fmt.Errorf("loading DMS %s: %w", loader.name, err)
		}
	}
	return nil
}

func dmsOptionalResourceErrorSkippable(err error) bool {
	var notFound *dmstypes.ResourceNotFoundFault
	if errors.As(err, &notFound) {
		return true
	}
	var accessDenied *dmstypes.AccessDeniedFault
	if errors.As(err, &accessDenied) {
		return true
	}
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && strings.Contains(strings.ToLower(apiErr.ErrorCode()), "accessdenied")
}

func (g *DmsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := databasemigrationservice.NewFromConfig(config)

	if err := g.loadReplicationInstances(svc); err != nil {
		return err
	}
	if err := g.loadReplicationSubnetGroups(svc); err != nil {
		return err
	}
	if err := g.loadEndpoints(svc); err != nil {
		return err
	}
	if err := g.loadReplicationTasks(svc); err != nil {
		return err
	}
	if err := g.loadOptionalResources([]dmsOptionalResourceLoader{
		{name: "certificates", load: func() error { return g.loadCertificates(svc) }},
		{name: "event subscriptions", load: func() error { return g.loadEventSubscriptions(svc) }},
		{name: "replication configs", load: func() error { return g.loadReplicationConfigs(svc) }},
	}); err != nil {
		return err
	}

	return nil
}

func (g *DmsGenerator) loadReplicationInstances(svc *databasemigrationservice.Client) error {
	p := databasemigrationservice.NewDescribeReplicationInstancesPaginator(svc, &databasemigrationservice.DescribeReplicationInstancesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, instance := range page.ReplicationInstances {
			if resource, ok := newDMSReplicationInstanceResource(instance); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *DmsGenerator) loadReplicationSubnetGroups(svc *databasemigrationservice.Client) error {
	p := databasemigrationservice.NewDescribeReplicationSubnetGroupsPaginator(svc, &databasemigrationservice.DescribeReplicationSubnetGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, group := range page.ReplicationSubnetGroups {
			if resource, ok := newDMSReplicationSubnetGroupResource(group); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *DmsGenerator) loadEndpoints(svc *databasemigrationservice.Client) error {
	p := databasemigrationservice.NewDescribeEndpointsPaginator(svc, &databasemigrationservice.DescribeEndpointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpoint := range page.Endpoints {
			if resource, ok := newDMSEndpointResource(endpoint); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *DmsGenerator) loadReplicationTasks(svc *databasemigrationservice.Client) error {
	p := databasemigrationservice.NewDescribeReplicationTasksPaginator(svc, &databasemigrationservice.DescribeReplicationTasksInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, task := range page.ReplicationTasks {
			if resource, ok := newDMSReplicationTaskResource(task); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *DmsGenerator) loadCertificates(svc *databasemigrationservice.Client) error {
	p := databasemigrationservice.NewDescribeCertificatesPaginator(svc, &databasemigrationservice.DescribeCertificatesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, certificate := range page.Certificates {
			if resource, ok := newDMSCertificateResource(certificate); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *DmsGenerator) loadEventSubscriptions(svc *databasemigrationservice.Client) error {
	p := databasemigrationservice.NewDescribeEventSubscriptionsPaginator(svc, &databasemigrationservice.DescribeEventSubscriptionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, subscription := range page.EventSubscriptionsList {
			if resource, ok := newDMSEventSubscriptionResource(subscription); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *DmsGenerator) loadReplicationConfigs(svc *databasemigrationservice.Client) error {
	p := databasemigrationservice.NewDescribeReplicationConfigsPaginator(svc, &databasemigrationservice.DescribeReplicationConfigsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, config := range page.ReplicationConfigs {
			if resource, ok := newDMSReplicationConfigResource(config); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newDMSCertificateResource(certificate dmstypes.Certificate) (terraformutils.Resource, bool) {
	identifier := StringValue(certificate.CertificateIdentifier)
	if identifier == "" || !dmsCertificateImportable(certificate) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		identifier,
		dmsResourceName("certificate", identifier),
		dmsCertificateResourceType,
		"aws",
		dmsAllowEmptyValues,
	), true
}

func newDMSEventSubscriptionResource(subscription dmstypes.EventSubscription) (terraformutils.Resource, bool) {
	name := StringValue(subscription.CustSubscriptionId)
	if name == "" || !dmsEventSubscriptionImportable(subscription) {
		return terraformutils.Resource{}, false
	}
	attributes := dmsStringSliceAttributes("event_categories", subscription.EventCategoriesList)
	if !subscription.Enabled {
		attributes["enabled"] = strconv.FormatBool(subscription.Enabled)
	}
	additionalFields := map[string]interface{}{}
	if len(subscription.EventCategoriesList) == 0 {
		// DMS omits categories for all-category subscriptions, but Terraform still requires an explicit empty set.
		additionalFields["event_categories"] = []interface{}{}
	}
	return terraformutils.NewResource(
		name,
		dmsResourceName("event-subscription", name),
		dmsEventSubscriptionResourceType,
		"aws",
		attributes,
		dmsAllowEmptyValues,
		additionalFields,
	), true
}

func newDMSReplicationConfigResource(config dmstypes.ReplicationConfig) (terraformutils.Resource, bool) {
	importID := dmsReplicationConfigImportID(config)
	if importID == "" || !dmsReplicationConfigImportable(config) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		dmsReplicationConfigResourceName(config),
		dmsReplicationConfigResourceType,
		"aws",
		dmsReplicationConfigAttributes(config),
		dmsAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newDMSReplicationInstanceResource(instance dmstypes.ReplicationInstance) (terraformutils.Resource, bool) {
	identifier := StringValue(instance.ReplicationInstanceIdentifier)
	if identifier == "" || !dmsReplicationInstanceImportable(instance) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		identifier,
		dmsResourceName("replication-instance", identifier),
		dmsReplicationInstanceResourceType,
		"aws",
		dmsAllowEmptyValues,
	), true
}

func newDMSReplicationSubnetGroupResource(group dmstypes.ReplicationSubnetGroup) (terraformutils.Resource, bool) {
	identifier := StringValue(group.ReplicationSubnetGroupIdentifier)
	if identifier == "" || !dmsStableStatusImportable(StringValue(group.SubnetGroupStatus)) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		identifier,
		dmsResourceName("replication-subnet-group", identifier),
		dmsReplicationSubnetGroupResourceType,
		"aws",
		dmsAllowEmptyValues,
	), true
}

func newDMSEndpointResource(endpoint dmstypes.Endpoint) (terraformutils.Resource, bool) {
	identifier := StringValue(endpoint.EndpointIdentifier)
	if identifier == "" || !dmsEndpointImportable(endpoint) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		identifier,
		dmsResourceName("endpoint", identifier),
		dmsEndpointResourceType,
		"aws",
		dmsAllowEmptyValues,
	), true
}

func newDMSReplicationTaskResource(task dmstypes.ReplicationTask) (terraformutils.Resource, bool) {
	identifier := StringValue(task.ReplicationTaskIdentifier)
	if identifier == "" || !dmsReplicationTaskImportable(task) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		identifier,
		dmsResourceName("replication-task", identifier),
		dmsReplicationTaskResourceType,
		"aws",
		dmsAllowEmptyValues,
	), true
}

func dmsResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return "dms-resource"
	}
	return strings.Join(cleanParts, "/")
}

func dmsReplicationConfigImportID(config dmstypes.ReplicationConfig) string {
	return StringValue(config.ReplicationConfigArn)
}

func dmsReplicationConfigResourceName(config dmstypes.ReplicationConfig) string {
	arnSuffix := arnLastSegment(dmsReplicationConfigImportID(config), ":")
	return dmsResourceName("replication-config", StringValue(config.ReplicationConfigIdentifier), arnSuffix)
}

func dmsCertificateImportable(certificate dmstypes.Certificate) bool {
	return StringValue(certificate.CertificatePem) != "" || len(certificate.CertificateWallet) > 0
}

func dmsStringSliceAttributes(prefix string, values []string) map[string]string {
	attributes := map[string]string{
		prefix + ".#": strconv.Itoa(len(values)),
	}
	for i, value := range values {
		attributes[prefix+"."+strconv.Itoa(i)] = value
	}
	return attributes
}

func dmsEventSubscriptionImportable(subscription dmstypes.EventSubscription) bool {
	return strings.EqualFold(StringValue(subscription.Status), dmsEventSubscriptionStatusActive) &&
		dmsEventSubscriptionSourceTypeImportable(StringValue(subscription.SourceType)) &&
		StringValue(subscription.SnsTopicArn) != ""
}

func dmsEventSubscriptionSourceTypeImportable(sourceType string) bool {
	switch strings.ToLower(sourceType) {
	case dmsEventSubscriptionSourceTypeReplicationInstance, dmsEventSubscriptionSourceTypeReplicationTask:
		return true
	default:
		return false
	}
}

func dmsReplicationConfigImportable(config dmstypes.ReplicationConfig) bool {
	if config.IsReadOnly != nil && *config.IsReadOnly {
		return false
	}
	return StringValue(config.ReplicationConfigArn) != "" &&
		StringValue(config.ReplicationConfigIdentifier) != "" &&
		config.ReplicationType != "" &&
		StringValue(config.SourceEndpointArn) != "" &&
		StringValue(config.TargetEndpointArn) != "" &&
		StringValue(config.TableMappings) != "" &&
		dmsReplicationConfigComputeConfigImportable(config.ComputeConfig)
}

func dmsReplicationConfigComputeConfigImportable(computeConfig *dmstypes.ComputeConfig) bool {
	return computeConfig != nil &&
		StringValue(computeConfig.ReplicationSubnetGroupId) != "" &&
		computeConfig.MaxCapacityUnits != nil &&
		*computeConfig.MaxCapacityUnits > 0
}

func dmsReplicationConfigAttributes(config dmstypes.ReplicationConfig) map[string]string {
	attributes := map[string]string{
		"compute_config.#":              "1",
		"replication_config_identifier": StringValue(config.ReplicationConfigIdentifier),
		"replication_type":              string(config.ReplicationType),
		"source_endpoint_arn":           StringValue(config.SourceEndpointArn),
		"table_mappings":                StringValue(config.TableMappings),
		"target_endpoint_arn":           StringValue(config.TargetEndpointArn),
	}
	if settings := StringValue(config.ReplicationSettings); settings != "" {
		attributes["replication_settings"] = settings
	}
	if settings := StringValue(config.SupplementalSettings); settings != "" {
		attributes["supplemental_settings"] = settings
	}
	if config.ComputeConfig == nil {
		return attributes
	}

	computeConfig := config.ComputeConfig
	attributes["compute_config.0.replication_subnet_group_id"] = StringValue(computeConfig.ReplicationSubnetGroupId)
	attributes["compute_config.0.max_capacity_units"] = strconv.Itoa(int(*computeConfig.MaxCapacityUnits))
	if value := StringValue(computeConfig.AvailabilityZone); value != "" {
		attributes["compute_config.0.availability_zone"] = value
	}
	if value := StringValue(computeConfig.DnsNameServers); value != "" {
		attributes["compute_config.0.dns_name_servers"] = value
	}
	if value := StringValue(computeConfig.KmsKeyId); value != "" {
		attributes["compute_config.0.kms_key_id"] = value
	}
	if computeConfig.MinCapacityUnits != nil {
		attributes["compute_config.0.min_capacity_units"] = strconv.Itoa(int(*computeConfig.MinCapacityUnits))
	}
	if computeConfig.MultiAZ != nil {
		attributes["compute_config.0.multi_az"] = strconv.FormatBool(*computeConfig.MultiAZ)
	}
	if value := StringValue(computeConfig.PreferredMaintenanceWindow); value != "" {
		attributes["compute_config.0.preferred_maintenance_window"] = value
	}
	for key, value := range dmsStringSliceAttributes("compute_config.0.vpc_security_group_ids", computeConfig.VpcSecurityGroupIds) {
		attributes[key] = value
	}
	return attributes
}

func dmsReplicationInstanceImportable(instance dmstypes.ReplicationInstance) bool {
	return strings.EqualFold(StringValue(instance.ReplicationInstanceStatus), dmsReplicationInstanceStatusAvailable)
}

func dmsEndpointImportable(endpoint dmstypes.Endpoint) bool {
	if strings.EqualFold(StringValue(endpoint.EngineName), dmsEndpointEngineS3) {
		return false
	}
	return dmsStableStatusImportable(StringValue(endpoint.Status))
}

func dmsReplicationTaskImportable(task dmstypes.ReplicationTask) bool {
	switch strings.ToLower(StringValue(task.Status)) {
	case dmsReplicationTaskStatusReady, dmsReplicationTaskStatusRunning, dmsReplicationTaskStatusStopped:
		return true
	default:
		return false
	}
}

func dmsStableStatusImportable(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return false
	}
	unsafeFragments := []string{
		"creat",
		"delet",
		"fail",
		"modif",
		"moving",
		"starting",
		"stopping",
		"testing",
		"upgrading",
	}
	for _, fragment := range unsafeFragments {
		if strings.Contains(status, fragment) {
			return false
		}
	}
	return true
}
