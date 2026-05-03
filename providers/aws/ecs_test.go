// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func TestArnLastSegment(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  string
		want string
	}{
		{"ecs cluster arn", "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster", "/", "my-cluster"},
		{"ecs service arn", "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service", "/", "my-service"},
		{"sns topic arn", "arn:aws:sns:us-east-1:123456789012:my-topic", ":", "my-topic"},
		{"sqs queue url", "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue", "/", "my-queue"},
		{"sqs fifo url", "https://sqs.eu-west-1.amazonaws.com/987654321098/orders.fifo", "/", "orders.fifo"},
		{"no separator", "simple-string", "/", "simple-string"},
		{"empty string", "", "/", ""},
		{"trailing separator", "a/b/c/", "/", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := arnLastSegment(tc.s, tc.sep); got != tc.want {
				t.Errorf("arnLastSegment(%q, %q) = %q, want %q", tc.s, tc.sep, got, tc.want)
			}
		})
	}
}

func TestEcsTaskSetImportID(t *testing.T) {
	if got, want := ecsTaskSetImportID("task-set-id", "service", "cluster"), "task-set-id,service,cluster"; got != want {
		t.Fatalf("ecsTaskSetImportID() = %q, want %q", got, want)
	}
}

func TestEcsResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "joins parts", parts: []string{"cluster", "service"}, want: "cluster_service"},
		{name: "omits empty parts", parts: []string{"", "cluster", "", "service"}, want: "cluster_service"},
		{name: "empty", parts: []string{"", ""}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ecsResourceName(tt.parts...); got != tt.want {
				t.Fatalf("ecsResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEcsTaskDefinitionRevision(t *testing.T) {
	tests := []struct {
		name           string
		arn            string
		wantDefinition string
		wantRevision   int
		wantErr        string
	}{
		{
			name:           "valid",
			arn:            "arn:aws:ecs:us-east-1:123456789012:task-definition/example:42",
			wantDefinition: "task-definition/example",
			wantRevision:   42,
		},
		{
			name:    "missing revision",
			arn:     "task-definition/example",
			wantErr: "missing revision",
		},
		{
			name:    "non numeric revision",
			arn:     "arn:aws:ecs:us-east-1:123456789012:task-definition/example:not-a-number",
			wantErr: "parse ecs task definition revision",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition, revision, err := ecsTaskDefinitionRevision(tt.arn)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error: %v", err)
			}
			if definition != tt.wantDefinition {
				t.Fatalf("definition = %q, want %q", definition, tt.wantDefinition)
			}
			if revision != tt.wantRevision {
				t.Fatalf("revision = %d, want %d", revision, tt.wantRevision)
			}
		})
	}
}

func TestEcsCapacityProviderImportable(t *testing.T) {
	tests := []struct {
		name             string
		capacityProvider ecstypes.CapacityProvider
		want             bool
	}{
		{
			name: "auto scaling group provider",
			capacityProvider: ecstypes.CapacityProvider{
				AutoScalingGroupProvider: &ecstypes.AutoScalingGroupProvider{},
			},
			want: true,
		},
		{
			name: "managed instances provider",
			capacityProvider: ecstypes.CapacityProvider{
				ManagedInstancesProvider: &ecstypes.ManagedInstancesProvider{},
			},
			want: true,
		},
		{
			name:             "built in provider",
			capacityProvider: ecstypes.CapacityProvider{},
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ecsCapacityProviderImportable(tt.capacityProvider); got != tt.want {
				t.Fatalf("ecsCapacityProviderImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEcsServiceDetails(t *testing.T) {
	taskDefinition := "arn:aws:ecs:us-east-1:123456789012:task-definition/example:1"
	service, err := ecsServiceDetails(&ecs.DescribeServicesOutput{
		Services: []ecstypes.Service{
			{TaskDefinition: &taskDefinition},
		},
	}, "example")
	if err != nil {
		t.Fatalf("ecsServiceDetails returned error: %v", err)
	}
	if got := aws.ToString(service.TaskDefinition); got != taskDefinition {
		t.Fatalf("TaskDefinition = %q, want %q", got, taskDefinition)
	}
}

func TestEcsServiceDetailsReturnsFailureReason(t *testing.T) {
	_, err := ecsServiceDetails(&ecs.DescribeServicesOutput{
		Failures: []ecstypes.Failure{
			{
				Arn:    aws.String("arn:aws:ecs:us-east-1:123456789012:service/example-cluster/example-service"),
				Reason: aws.String("MISSING"),
			},
		},
	}, "example-service")
	if err == nil {
		t.Fatal("expected describe services failure error")
	}
	if !strings.Contains(err.Error(), "service example-service was not described: MISSING") {
		t.Fatalf("error = %q, want failure reason", err)
	}
}

func TestEcsServiceDetailsReturnsEmptyResponseError(t *testing.T) {
	_, err := ecsServiceDetails(nil, "example-service")
	if err == nil {
		t.Fatal("expected nil describe services response error")
	}
	if !strings.Contains(err.Error(), "empty describe services response") {
		t.Fatalf("error = %q, want empty response context", err)
	}
}

func TestEcsClusterNotFound(t *testing.T) {
	if !ecsClusterNotFound(&ecstypes.ClusterNotFoundException{}) {
		t.Fatal("ecsClusterNotFound() = false for ClusterNotFoundException, want true")
	}
	if ecsClusterNotFound(errors.New("boom")) {
		t.Fatal("ecsClusterNotFound() = true for generic error, want false")
	}
	if ecsClusterNotFound(nil) {
		t.Fatal("ecsClusterNotFound() = true for nil, want false")
	}
}

func TestEcsTaskSetScopeNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "cluster missing", err: &ecstypes.ClusterNotFoundException{}, want: true},
		{name: "service missing", err: &ecstypes.ServiceNotFoundException{}, want: true},
		{name: "task set missing", err: &ecstypes.TaskSetNotFoundException{}, want: true},
		{name: "generic error", err: errors.New("boom"), want: false},
		{name: "nil", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ecsTaskSetScopeNotFound(tt.err); got != tt.want {
				t.Fatalf("ecsTaskSetScopeNotFound() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEcsTaskSetUnsupported(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "invalid parameter", err: &ecstypes.InvalidParameterException{}, want: true},
		{name: "client exception", err: &ecstypes.ClientException{}, want: true},
		{name: "unsupported feature", err: &ecstypes.UnsupportedFeatureException{}, want: true},
		{name: "generic error", err: errors.New("boom"), want: false},
		{name: "nil", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ecsTaskSetUnsupported(tt.err); got != tt.want {
				t.Fatalf("ecsTaskSetUnsupported() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEcsTaskSetDiscoverySkipError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "scope missing", err: &ecstypes.ServiceNotFoundException{}, want: true},
		{name: "unsupported service", err: &ecstypes.InvalidParameterException{}, want: true},
		{name: "generic error", err: errors.New("boom"), want: false},
		{name: "nil", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ecsTaskSetDiscoverySkipError(tt.err); got != tt.want {
				t.Fatalf("ecsTaskSetDiscoverySkipError() = %t, want %t", got, tt.want)
			}
		})
	}
}
