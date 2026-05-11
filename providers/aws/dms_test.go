// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	dmstypes "github.com/aws/aws-sdk-go-v2/service/databasemigrationservice/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestDMSResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "filters empty parts", parts: []string{"", "endpoint", "", "orders"}, want: "endpoint/orders"},
		{name: "preserves segment boundaries", parts: []string{"replication-task", "orders"}, want: "replication-task/orders"},
		{name: "fallback", parts: nil, want: "dms-resource"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dmsResourceName(tt.parts...)
			if got != tt.want {
				t.Fatalf("dmsResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDMSResourceNamesAvoidTypeCollisions(t *testing.T) {
	endpoint := dmsResourceName("endpoint", "shared")
	task := dmsResourceName("replication-task", "shared")
	if endpoint == task {
		t.Fatalf("resource names collide: %q", endpoint)
	}
}

func TestDMSOptionalResourceLoaderErrors(t *testing.T) {
	t.Run("skips resource not found", func(t *testing.T) {
		g := DmsGenerator{}
		calledNext := false
		err := g.loadOptionalResources([]dmsOptionalResourceLoader{
			{
				name: "certificates",
				load: func() error {
					return &dmstypes.ResourceNotFoundFault{}
				},
			},
			{
				name: "event subscriptions",
				load: func() error {
					calledNext = true
					return nil
				},
			},
		})
		if err != nil {
			t.Fatalf("loadOptionalResources() error = %v, want nil", err)
		}
		if !calledNext {
			t.Fatal("loadOptionalResources() should continue after skippable error")
		}
	})

	t.Run("returns unexpected error", func(t *testing.T) {
		g := DmsGenerator{}
		boom := errors.New("boom")
		calledNext := false
		err := g.loadOptionalResources([]dmsOptionalResourceLoader{
			{
				name: "certificates",
				load: func() error {
					return boom
				},
			},
			{
				name: "event subscriptions",
				load: func() error {
					calledNext = true
					return nil
				},
			},
		})
		if !errors.Is(err, boom) {
			t.Fatalf("loadOptionalResources() error = %v, want %v", err, boom)
		}
		if calledNext {
			t.Fatal("loadOptionalResources() should stop after unexpected error")
		}
	})
}

func TestNewDMSCertificateResource(t *testing.T) {
	resource, ok := newDMSCertificateResource(dmstypes.Certificate{
		CertificateIdentifier: dmsString("dms-cert"),
		CertificatePem:        dmsString("-----BEGIN CERTIFICATE-----"),
	})
	assertDMSResource(t, resource, ok, "dms-cert", dmsResourceName("certificate", "dms-cert"), dmsCertificateResourceType)

	if _, ok := newDMSCertificateResource(dmstypes.Certificate{CertificatePem: dmsString("-----BEGIN CERTIFICATE-----")}); ok {
		t.Fatal("certificate with empty identifier should be skipped")
	}
	if _, ok := newDMSCertificateResource(dmstypes.Certificate{CertificateIdentifier: dmsString("dms-cert")}); ok {
		t.Fatal("certificate without PEM or wallet material should be skipped")
	}
}

func TestNewDMSEventSubscriptionResource(t *testing.T) {
	resource, ok := newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		CustSubscriptionId:  dmsString("dms-events"),
		EventCategoriesList: []string{"creation", "failure"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("replication-task"),
		Status:              dmsString("active"),
	})
	assertDMSResource(t, resource, ok, "dms-events", dmsResourceName("event-subscription", "dms-events"), dmsEventSubscriptionResourceType)

	resource, ok = newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		CustSubscriptionId: dmsString("dms-all-events"),
		SnsTopicArn:        dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:         dmsString("replication-instance"),
		Status:             dmsString("active"),
	})
	assertDMSResource(t, resource, ok, "dms-all-events", dmsResourceName("event-subscription", "dms-all-events"), dmsEventSubscriptionResourceType)

	if _, ok := newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		EventCategoriesList: []string{"creation"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("replication-task"),
		Status:              dmsString("active"),
	}); ok {
		t.Fatal("event subscription with empty name should be skipped")
	}
	if _, ok := newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		CustSubscriptionId:  dmsString("dms-events"),
		EventCategoriesList: []string{"creation"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("replication-task"),
		Status:              dmsString("creating"),
	}); ok {
		t.Fatal("creating event subscription should be skipped")
	}
	if _, ok := newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		CustSubscriptionId:  dmsString("dms-events"),
		EventCategoriesList: []string{"creation"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("security-group"),
		Status:              dmsString("active"),
	}); ok {
		t.Fatal("event subscription with unsupported source type should be skipped")
	}
}

func TestNewDMSReplicationInstanceResource(t *testing.T) {
	resource, ok := newDMSReplicationInstanceResource(dmstypes.ReplicationInstance{
		ReplicationInstanceIdentifier: dmsString("dms-ri"),
		ReplicationInstanceStatus:     dmsString("available"),
	})
	assertDMSResource(t, resource, ok, "dms-ri", dmsResourceName("replication-instance", "dms-ri"), dmsReplicationInstanceResourceType)

	if _, ok := newDMSReplicationInstanceResource(dmstypes.ReplicationInstance{ReplicationInstanceStatus: dmsString("available")}); ok {
		t.Fatal("replication instance with empty identifier should be skipped")
	}
	if _, ok := newDMSReplicationInstanceResource(dmstypes.ReplicationInstance{
		ReplicationInstanceIdentifier: dmsString("dms-ri"),
		ReplicationInstanceStatus:     dmsString("creating"),
	}); ok {
		t.Fatal("creating replication instance should be skipped")
	}
}

func TestNewDMSReplicationSubnetGroupResource(t *testing.T) {
	resource, ok := newDMSReplicationSubnetGroupResource(dmstypes.ReplicationSubnetGroup{
		ReplicationSubnetGroupIdentifier: dmsString("dms-subnets"),
		SubnetGroupStatus:                dmsString("Complete"),
	})
	assertDMSResource(t, resource, ok, "dms-subnets", dmsResourceName("replication-subnet-group", "dms-subnets"), dmsReplicationSubnetGroupResourceType)

	if _, ok := newDMSReplicationSubnetGroupResource(dmstypes.ReplicationSubnetGroup{SubnetGroupStatus: dmsString("Complete")}); ok {
		t.Fatal("replication subnet group with empty identifier should be skipped")
	}
	if _, ok := newDMSReplicationSubnetGroupResource(dmstypes.ReplicationSubnetGroup{
		ReplicationSubnetGroupIdentifier: dmsString("dms-subnets"),
		SubnetGroupStatus:                dmsString("deleting"),
	}); ok {
		t.Fatal("deleting replication subnet group should be skipped")
	}
}

func TestNewDMSEndpointResource(t *testing.T) {
	resource, ok := newDMSEndpointResource(dmstypes.Endpoint{
		EndpointIdentifier: dmsString("source-endpoint"),
		EngineName:         dmsString("postgres"),
		Status:             dmsString("active"),
	})
	assertDMSResource(t, resource, ok, "source-endpoint", dmsResourceName("endpoint", "source-endpoint"), dmsEndpointResourceType)

	if _, ok := newDMSEndpointResource(dmstypes.Endpoint{Status: dmsString("active")}); ok {
		t.Fatal("endpoint with empty identifier should be skipped")
	}
	if _, ok := newDMSEndpointResource(dmstypes.Endpoint{
		EndpointIdentifier: dmsString("s3-target"),
		EngineName:         dmsString("s3"),
		Status:             dmsString("active"),
	}); ok {
		t.Fatal("S3 endpoint should be skipped for aws_dms_endpoint")
	}
	if _, ok := newDMSEndpointResource(dmstypes.Endpoint{
		EndpointIdentifier: dmsString("source-endpoint"),
		EngineName:         dmsString("postgres"),
		Status:             dmsString("failed"),
	}); ok {
		t.Fatal("failed endpoint should be skipped")
	}
}

func TestNewDMSReplicationTaskResource(t *testing.T) {
	resource, ok := newDMSReplicationTaskResource(dmstypes.ReplicationTask{
		ReplicationTaskIdentifier: dmsString("orders-task"),
		Status:                    dmsString("ready"),
	})
	assertDMSResource(t, resource, ok, "orders-task", dmsResourceName("replication-task", "orders-task"), dmsReplicationTaskResourceType)

	if _, ok := newDMSReplicationTaskResource(dmstypes.ReplicationTask{Status: dmsString("ready")}); ok {
		t.Fatal("replication task with empty identifier should be skipped")
	}
	if _, ok := newDMSReplicationTaskResource(dmstypes.ReplicationTask{
		ReplicationTaskIdentifier: dmsString("orders-task"),
		Status:                    dmsString("creating"),
	}); ok {
		t.Fatal("creating replication task should be skipped")
	}
}

func TestDMSCertificateImportable(t *testing.T) {
	if !dmsCertificateImportable(dmstypes.Certificate{CertificatePem: dmsString("pem")}) {
		t.Fatal("certificate with PEM material should be importable")
	}
	if !dmsCertificateImportable(dmstypes.Certificate{CertificateWallet: []byte("wallet")}) {
		t.Fatal("certificate with wallet material should be importable")
	}
	if dmsCertificateImportable(dmstypes.Certificate{}) {
		t.Fatal("certificate without PEM or wallet material should not be importable")
	}
}

func TestDMSEventSubscriptionImportable(t *testing.T) {
	base := dmstypes.EventSubscription{
		EventCategoriesList: []string{"creation"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("replication-instance"),
		Status:              dmsString("active"),
	}
	if !dmsEventSubscriptionImportable(base) {
		t.Fatal("active event subscription with supported source type and required fields should be importable")
	}
	allCategories := base
	allCategories.EventCategoriesList = nil
	if !dmsEventSubscriptionImportable(allCategories) {
		t.Fatal("active event subscription without event categories should import all categories")
	}
	caseVariant := base
	caseVariant.SourceType = dmsString("REPLICATION-TASK")
	caseVariant.Status = dmsString("ACTIVE")
	if !dmsEventSubscriptionImportable(caseVariant) {
		t.Fatal("event subscription importability should be case-insensitive for status and source type")
	}

	tests := []struct {
		name         string
		subscription dmstypes.EventSubscription
	}{
		{name: "empty status", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
			SourceType:          dmsString("replication-instance"),
		}},
		{name: "creating status", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
			SourceType:          dmsString("replication-instance"),
			Status:              dmsString("creating"),
		}},
		{name: "no permission status", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
			SourceType:          dmsString("replication-instance"),
			Status:              dmsString("no-permission"),
		}},
		{name: "unsupported source type", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
			SourceType:          dmsString("security-group"),
			Status:              dmsString("active"),
		}},
		{name: "empty sns topic arn", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SourceType:          dmsString("replication-instance"),
			Status:              dmsString("active"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if dmsEventSubscriptionImportable(tt.subscription) {
				t.Fatal("event subscription should not be importable")
			}
		})
	}
}

func TestDMSEventSubscriptionSourceTypeImportable(t *testing.T) {
	for _, sourceType := range []string{"replication-instance", "replication-task", "REPLICATION-INSTANCE", "RePlIcAtIoN-TaSk"} {
		if !dmsEventSubscriptionSourceTypeImportable(sourceType) {
			t.Fatalf("source type %q should be importable", sourceType)
		}
	}
	for _, sourceType := range []string{"", "replication-server", "security-group"} {
		if dmsEventSubscriptionSourceTypeImportable(sourceType) {
			t.Fatalf("source type %q should not be importable", sourceType)
		}
	}
}

func TestDMSReplicationInstanceImportable(t *testing.T) {
	if !dmsReplicationInstanceImportable(dmstypes.ReplicationInstance{ReplicationInstanceStatus: dmsString("available")}) {
		t.Fatal("available replication instance should be importable")
	}
	for _, status := range []string{"creating", "deleting", "failed", "modifying", "upgrading", ""} {
		if dmsReplicationInstanceImportable(dmstypes.ReplicationInstance{ReplicationInstanceStatus: dmsString(status)}) {
			t.Fatalf("replication instance status %q should not be importable", status)
		}
	}
}

func TestDMSEndpointImportable(t *testing.T) {
	if !dmsEndpointImportable(dmstypes.Endpoint{EngineName: dmsString("postgres"), Status: dmsString("active")}) {
		t.Fatal("active non-S3 endpoint should be importable")
	}
	if dmsEndpointImportable(dmstypes.Endpoint{EngineName: dmsString("s3"), Status: dmsString("active")}) {
		t.Fatal("S3 endpoint should not be imported as aws_dms_endpoint")
	}
	if dmsEndpointImportable(dmstypes.Endpoint{EngineName: dmsString("postgres"), Status: dmsString("failed")}) {
		t.Fatal("failed endpoint should not be importable")
	}
	if dmsEndpointImportable(dmstypes.Endpoint{EngineName: dmsString("postgres")}) {
		t.Fatal("endpoint with empty status should not be importable")
	}
}

func TestDMSReplicationTaskImportable(t *testing.T) {
	for _, status := range []string{"ready", "running", "stopped"} {
		if !dmsReplicationTaskImportable(dmstypes.ReplicationTask{Status: dmsString(status)}) {
			t.Fatalf("replication task status %q should be importable", status)
		}
	}
	for _, status := range []string{"creating", "deleting", "failed", "failed-move", "modifying", "moving", "starting", "stopping", ""} {
		if dmsReplicationTaskImportable(dmstypes.ReplicationTask{Status: dmsString(status)}) {
			t.Fatalf("replication task status %q should not be importable", status)
		}
	}
}

func TestDMSStableStatusImportable(t *testing.T) {
	for _, status := range []string{"active", "available", "Complete", "ready", "running", "stopped"} {
		if !dmsStableStatusImportable(status) {
			t.Fatalf("status %q should be importable", status)
		}
	}
	for _, status := range []string{"", "creating", "deleting", "failed", "failed-move", "modifying", "moving", "starting", "stopping", "testing", "upgrading"} {
		if dmsStableStatusImportable(status) {
			t.Fatalf("status %q should not be importable", status)
		}
	}
}

func assertDMSResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want %q", got, terraformutils.TfSanitize(wantName))
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
}

func dmsString(value string) *string {
	return &value
}
