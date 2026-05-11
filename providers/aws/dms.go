// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	dmstypes "github.com/aws/aws-sdk-go-v2/service/databasemigrationservice/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	dmsCertificateResourceType            = "aws_dms_certificate"
	dmsEventSubscriptionResourceType      = "aws_dms_event_subscription"
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

var dmsAllowEmptyValues = []string{"tags."}

type DmsGenerator struct {
	AWSService
}

type dmsOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *DmsGenerator) loadOptionalResources(loaders []dmsOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("Skipping DMS %s: %v", loader.name, err)
		}
	}
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
	g.loadOptionalResources([]dmsOptionalResourceLoader{
		{name: "certificates", load: func() error { return g.loadCertificates(svc) }},
		{name: "event subscriptions", load: func() error { return g.loadEventSubscriptions(svc) }},
	})

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
	return terraformutils.NewSimpleResource(
		name,
		dmsResourceName("event-subscription", name),
		dmsEventSubscriptionResourceType,
		"aws",
		dmsAllowEmptyValues,
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

func dmsCertificateImportable(certificate dmstypes.Certificate) bool {
	return StringValue(certificate.CertificatePem) != "" || len(certificate.CertificateWallet) > 0
}

func dmsEventSubscriptionImportable(subscription dmstypes.EventSubscription) bool {
	return strings.EqualFold(StringValue(subscription.Status), dmsEventSubscriptionStatusActive) &&
		dmsEventSubscriptionSourceTypeImportable(StringValue(subscription.SourceType)) &&
		StringValue(subscription.SnsTopicArn) != "" &&
		len(subscription.EventCategoriesList) > 0
}

func dmsEventSubscriptionSourceTypeImportable(sourceType string) bool {
	switch strings.ToLower(sourceType) {
	case dmsEventSubscriptionSourceTypeReplicationInstance, dmsEventSubscriptionSourceTypeReplicationTask:
		return true
	default:
		return false
	}
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
