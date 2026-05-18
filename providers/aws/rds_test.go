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
		{FeatureName: aws.String("s3Import"), RoleArn: aws.String(roleARN), Status: aws.String("PENDING")},
		{FeatureName: aws.String("s3Import"), RoleArn: aws.String(roleARN), Status: aws.String("ACTIVE")},
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

func TestRDSClusterInstanceResource(t *testing.T) {
	resource := newRDSClusterInstanceResource("db-1", "cluster-1")
	if got, want := resource.InstanceInfo.Type, "aws_rds_cluster_instance"; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.ID, "db-1"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.ResourceName, "tfer--9_cluster-1__4_db-1"; got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["cluster_identifier"], "cluster-1"; got != want {
		t.Fatalf("cluster_identifier = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["identifier"], "db-1"; got != want {
		t.Fatalf("identifier = %q, want %q", got, want)
	}
}

func TestRDSAddClusterRoleAssociations(t *testing.T) {
	roleARN := "arn:aws:iam::123456789012:role/service-role/rds-cluster-role"
	g := &RDSGenerator{}
	g.addRDSClusterRoleAssociations("cluster-1", []rdstypes.DBClusterRole{
		{FeatureName: aws.String("s3Import"), RoleArn: aws.String(roleARN), Status: aws.String("PENDING")},
		{FeatureName: aws.String("s3Import"), RoleArn: aws.String(roleARN), Status: aws.String("ACTIVE")},
	})

	if len(g.Resources) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(g.Resources))
	}
	resource := g.Resources[0]
	if got, want := resource.InstanceInfo.Type, "aws_rds_cluster_role_association"; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.ID, "cluster-1,"+roleARN; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["db_cluster_identifier"], "cluster-1"; got != want {
		t.Fatalf("db_cluster_identifier = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["feature_name"], "s3Import"; got != want {
		t.Fatalf("feature_name = %q, want %q", got, want)
	}
}

func TestRDSAddClusterActivityStream(t *testing.T) {
	clusterARN := "arn:aws:rds:us-east-1:123456789012:cluster:cluster-1"
	g := &RDSGenerator{}
	g.addRDSClusterActivityStream(rdstypes.DBCluster{
		ActivityStreamKmsKeyId: aws.String("arn:aws:kms:us-east-1:123456789012:key/key-id"),
		ActivityStreamMode:     rdstypes.ActivityStreamModeAsync,
		ActivityStreamStatus:   rdstypes.ActivityStreamStatusStarted,
		DBClusterArn:           aws.String(clusterARN),
		DBClusterIdentifier:    aws.String("cluster-1"),
	})
	g.addRDSClusterActivityStream(rdstypes.DBCluster{
		ActivityStreamMode:   rdstypes.ActivityStreamModeAsync,
		ActivityStreamStatus: rdstypes.ActivityStreamStatusStopped,
		DBClusterArn:         aws.String("arn:aws:rds:us-east-1:123456789012:cluster:cluster-2"),
	})

	if len(g.Resources) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(g.Resources))
	}
	resource := g.Resources[0]
	if got, want := resource.InstanceInfo.Type, "aws_rds_cluster_activity_stream"; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.ID, clusterARN; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["resource_arn"], clusterARN; got != want {
		t.Fatalf("resource_arn = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["mode"], "async"; got != want {
		t.Fatalf("mode = %q, want %q", got, want)
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

func TestRDSCompositeResourceName(t *testing.T) {
	if got, want := rdsCompositeResourceName(), "rds_resource"; got != want {
		t.Fatalf("rdsCompositeResourceName() = %q, want %q", got, want)
	}
	if got, want := rdsCompositeResourceName("cluster-1", "db-1"), "9_cluster-1__4_db-1"; got != want {
		t.Fatalf("rdsCompositeResourceName() = %q, want %q", got, want)
	}
	first := rdsCompositeResourceName("ab", "c")
	second := rdsCompositeResourceName("a", "bc")
	if first == second {
		t.Fatalf("rdsCompositeResourceName collision: %q == %q", first, second)
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

func TestRDSRoleAssociationStatusImportable(t *testing.T) {
	if !rdsRoleAssociationStatusImportable("ACTIVE") {
		t.Fatal("ACTIVE role association should be importable")
	}
	if rdsRoleAssociationStatusImportable("PENDING") {
		t.Fatal("PENDING role association should be skipped")
	}
	if rdsRoleAssociationStatusImportable("") {
		t.Fatal("empty role association status should be skipped")
	}
}

func TestRDSActivityStreamStatusImportable(t *testing.T) {
	if !rdsActivityStreamStatusImportable(rdstypes.ActivityStreamStatusStarted) {
		t.Fatal("started activity stream should be importable")
	}
	if rdsActivityStreamStatusImportable(rdstypes.ActivityStreamStatusStopped) {
		t.Fatal("stopped activity stream should be skipped")
	}
}
