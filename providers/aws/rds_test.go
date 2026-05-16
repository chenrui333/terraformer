// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
)

func TestRDSDBProxyEndpointImportID(t *testing.T) {
	got := rdsDBProxyEndpointImportID("proxy", "endpoint")
	want := "proxy/endpoint"
	if got != want {
		t.Fatalf("rdsDBProxyEndpointImportID() = %q, want %q", got, want)
	}
}

func TestRDSDBProxyTargetImportID(t *testing.T) {
	got := rdsDBProxyTargetImportID("proxy", "default", "RDS_INSTANCE", "database-1")
	want := "proxy/default/RDS_INSTANCE/database-1"
	if got != want {
		t.Fatalf("rdsDBProxyTargetImportID() = %q, want %q", got, want)
	}
}

func TestRDSRoleAssociationImportID(t *testing.T) {
	got := rdsRoleAssociationImportID("cluster-1", "arn:aws:iam::123456789012:role/service-role/rds")
	want := "cluster-1,arn:aws:iam::123456789012:role/service-role/rds"
	if got != want {
		t.Fatalf("rdsRoleAssociationImportID() = %q, want %q", got, want)
	}
}

func TestRDSAddDBInstanceRoleAssociations(t *testing.T) {
	roleARN := "arn:aws:iam::123456789012:role/service-role/rds-monitoring-role"
	g := &RDSGenerator{}
	g.addDBInstanceRoleAssociations("db-1", []rdstypes.DBInstanceRole{
		{FeatureName: aws.String("s3Import")},
		{RoleArn: aws.String(roleARN)},
		{FeatureName: aws.String("s3Import"), RoleArn: aws.String(roleARN)},
	})

	if len(g.Resources) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(g.Resources))
	}
	resource := g.Resources[0]
	if got, want := resource.InstanceInfo.Type, "aws_db_instance_role_association"; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.ID, "db-1,"+roleARN; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["feature_name"], "s3Import"; got != want {
		t.Fatalf("feature_name = %q, want %q", got, want)
	}
}

func TestRDSResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "filters empty parts", parts: []string{"", "cluster", "", "endpoint"}, want: "cluster_endpoint"},
		{name: "fallback", parts: nil, want: "rds_resource"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rdsResourceName(tt.parts...)
			if got != tt.want {
				t.Fatalf("rdsResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRDSIAMRoleResourceName(t *testing.T) {
	tests := []struct {
		name string
		arn  string
		want string
	}{
		{
			name: "role with path",
			arn:  "arn:aws:iam::123456789012:role/service-role/rds-monitoring-role",
			want: "service-role/rds-monitoring-role",
		},
		{
			name: "plain role",
			arn:  "arn:aws:iam::123456789012:role/rds-monitoring-role",
			want: "rds-monitoring-role",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rdsIAMRoleResourceName(tt.arn)
			if got != tt.want {
				t.Fatalf("rdsIAMRoleResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRDSCustomClusterEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint rdstypes.DBClusterEndpoint
		want     bool
	}{
		{name: "custom", endpoint: rdstypes.DBClusterEndpoint{EndpointType: aws.String("CUSTOM")}, want: true},
		{name: "reader", endpoint: rdstypes.DBClusterEndpoint{EndpointType: aws.String("READER")}, want: false},
		{name: "empty", endpoint: rdstypes.DBClusterEndpoint{}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rdsCustomClusterEndpoint(tt.endpoint)
			if got != tt.want {
				t.Fatalf("rdsCustomClusterEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRDSDefaultDBProxyEndpoint(t *testing.T) {
	if !rdsDefaultDBProxyEndpoint(rdstypes.DBProxyEndpoint{IsDefault: aws.Bool(true)}) {
		t.Fatal("rdsDefaultDBProxyEndpoint() = false, want true")
	}
	if rdsDefaultDBProxyEndpoint(rdstypes.DBProxyEndpoint{IsDefault: aws.Bool(false)}) {
		t.Fatal("rdsDefaultDBProxyEndpoint() = true, want false")
	}
	if rdsDefaultDBProxyEndpoint(rdstypes.DBProxyEndpoint{}) {
		t.Fatal("rdsDefaultDBProxyEndpoint() = true for nil flag, want false")
	}
}

func TestRDSDefaultDBProxyTargetGroup(t *testing.T) {
	if !rdsDefaultDBProxyTargetGroup(rdstypes.DBProxyTargetGroup{IsDefault: aws.Bool(true)}) {
		t.Fatal("rdsDefaultDBProxyTargetGroup() = false, want true")
	}
	if rdsDefaultDBProxyTargetGroup(rdstypes.DBProxyTargetGroup{IsDefault: aws.Bool(false)}) {
		t.Fatal("rdsDefaultDBProxyTargetGroup() = true, want false")
	}
	if rdsDefaultDBProxyTargetGroup(rdstypes.DBProxyTargetGroup{}) {
		t.Fatal("rdsDefaultDBProxyTargetGroup() = true for nil flag, want false")
	}
}

func TestRDSStatusImportable(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"available", true},
		{"backing-up", true},
		{"stopped", true},
		{"deleting", false},
		{"creating", false},
		{"failed", false},
		{"migration-failed", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := rdsStatusImportable(tt.status); got != tt.want {
				t.Fatalf("rdsStatusImportable(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
