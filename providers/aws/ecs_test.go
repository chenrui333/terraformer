// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

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
