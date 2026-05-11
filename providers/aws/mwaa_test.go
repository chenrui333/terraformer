// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	mwaatypes "github.com/aws/aws-sdk-go-v2/service/mwaa/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewMwaaEnvironmentResource(t *testing.T) {
	environment := validMwaaEnvironment("prod-airflow")
	resource, ok := newMwaaEnvironmentResource(environment)
	assertMwaaResource(t, resource, ok, "prod-airflow", mwaaResourceName("environment", "prod-airflow"), mwaaEnvironmentResourceType)
	assertMwaaAttribute(t, resource, "dag_s3_path", "dags")
	assertMwaaAttribute(t, resource, "execution_role_arn", "arn:aws:iam::123456789012:role/mwaa-execution")
	assertMwaaAttribute(t, resource, "name", "prod-airflow")
	assertMwaaAttribute(t, resource, "source_bucket_arn", "arn:aws:s3:::mwaa-source")
}

func TestNewMwaaEnvironmentResourceSkipsIncompleteEnvironments(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*mwaatypes.Environment)
	}{
		{
			name: "nil environment",
			mutate: func(environment *mwaatypes.Environment) {
				*environment = mwaatypes.Environment{}
			},
		},
		{
			name: "empty name",
			mutate: func(environment *mwaatypes.Environment) {
				environment.Name = nil
			},
		},
		{
			name: "empty DAG path",
			mutate: func(environment *mwaatypes.Environment) {
				environment.DagS3Path = nil
			},
		},
		{
			name: "empty execution role ARN",
			mutate: func(environment *mwaatypes.Environment) {
				environment.ExecutionRoleArn = nil
			},
		},
		{
			name: "empty source bucket ARN",
			mutate: func(environment *mwaatypes.Environment) {
				environment.SourceBucketArn = nil
			},
		},
		{
			name: "missing network configuration",
			mutate: func(environment *mwaatypes.Environment) {
				environment.NetworkConfiguration = nil
			},
		},
		{
			name: "empty security groups",
			mutate: func(environment *mwaatypes.Environment) {
				environment.NetworkConfiguration.SecurityGroupIds = nil
			},
		},
		{
			name: "single subnet",
			mutate: func(environment *mwaatypes.Environment) {
				environment.NetworkConfiguration.SubnetIds = []string{"subnet-1"}
			},
		},
		{
			name: "missing last update",
			mutate: func(environment *mwaatypes.Environment) {
				environment.LastUpdate = nil
			},
		},
		{
			name: "empty status",
			mutate: func(environment *mwaatypes.Environment) {
				environment.Status = ""
			},
		},
		{
			name: "deleting status",
			mutate: func(environment *mwaatypes.Environment) {
				environment.Status = mwaatypes.EnvironmentStatusDeleting
			},
		},
		{
			name: "deleted status",
			mutate: func(environment *mwaatypes.Environment) {
				environment.Status = mwaatypes.EnvironmentStatusDeleted
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment := validMwaaEnvironment("prod-airflow")
			tt.mutate(environment)
			if _, ok := newMwaaEnvironmentResource(environment); ok {
				t.Fatal("incomplete environment should be skipped")
			}
		})
	}

	if _, ok := newMwaaEnvironmentResource(nil); ok {
		t.Fatal("nil environment should be skipped")
	}
}

func TestMwaaEnvironmentImportID(t *testing.T) {
	got := mwaaEnvironmentImportID("prod-airflow")
	want := "prod-airflow"
	if got != want {
		t.Fatalf("environment import ID = %q, want %q", got, want)
	}
}

func TestMwaaEnvironmentStatusImportable(t *testing.T) {
	tests := []struct {
		name   string
		status mwaatypes.EnvironmentStatus
		want   bool
	}{
		{name: "creating", status: mwaatypes.EnvironmentStatusCreating, want: true},
		{name: "create failed", status: mwaatypes.EnvironmentStatusCreateFailed, want: true},
		{name: "available", status: mwaatypes.EnvironmentStatusAvailable, want: true},
		{name: "updating", status: mwaatypes.EnvironmentStatusUpdating, want: true},
		{name: "unavailable", status: mwaatypes.EnvironmentStatusUnavailable, want: true},
		{name: "update failed", status: mwaatypes.EnvironmentStatusUpdateFailed, want: true},
		{name: "rolling back", status: mwaatypes.EnvironmentStatusRollingBack, want: true},
		{name: "creating snapshot", status: mwaatypes.EnvironmentStatusCreatingSnapshot, want: true},
		{name: "pending", status: mwaatypes.EnvironmentStatusPending, want: true},
		{name: "maintenance", status: mwaatypes.EnvironmentStatusMaintenance, want: true},
		{name: "empty", want: false},
		{name: "deleting", status: mwaatypes.EnvironmentStatusDeleting, want: false},
		{name: "deleted", status: mwaatypes.EnvironmentStatusDeleted, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mwaaEnvironmentStatusImportable(tt.status); got != tt.want {
				t.Fatalf("mwaaEnvironmentStatusImportable(%q) = %t, want %t", tt.status, got, tt.want)
			}
		})
	}
}

func TestMwaaResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left, ok := newMwaaEnvironmentResource(validMwaaEnvironment("a/b_c"))
	if !ok {
		t.Fatal("left environment should be importable")
	}
	right, ok := newMwaaEnvironmentResource(validMwaaEnvironment("a-002F-b_c"))
	if !ok {
		t.Fatal("right environment should be importable")
	}
	if left.ResourceName == right.ResourceName {
		t.Fatalf("environment resource names collide: %q", left.ResourceName)
	}
}

func TestMwaaResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &mwaatypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &mwaatypes.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mwaaResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("mwaaResourceNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func assertMwaaResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
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

func assertMwaaAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}

func validMwaaEnvironment(name string) *mwaatypes.Environment {
	return &mwaatypes.Environment{
		DagS3Path:        mwaaString("dags"),
		ExecutionRoleArn: mwaaString("arn:aws:iam::123456789012:role/mwaa-execution"),
		LastUpdate:       &mwaatypes.LastUpdate{},
		Name:             mwaaString(name),
		NetworkConfiguration: &mwaatypes.NetworkConfiguration{
			SecurityGroupIds: []string{"sg-1"},
			SubnetIds:        []string{"subnet-1", "subnet-2"},
		},
		SourceBucketArn: mwaaString("arn:aws:s3:::mwaa-source"),
		Status:          mwaatypes.EnvironmentStatusAvailable,
	}
}

func mwaaString(value string) *string {
	return &value
}
