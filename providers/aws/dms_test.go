// SPDX-License-Identifier: Apache-2.0

package aws

import (
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
