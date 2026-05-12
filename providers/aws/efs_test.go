// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efstypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/smithy-go"
)

type terraformResourceResult struct {
	resource terraformutils.Resource
	ok       bool
}

func newTerraformResourceResult(resource terraformutils.Resource, ok bool) terraformResourceResult {
	return terraformResourceResult{resource: resource, ok: ok}
}

func TestEfsLoadFileSystemReturnsMountTargetErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2015-02-01/file-systems":
			writeEfsJSON(w, http.StatusOK, "{\"FileSystems\":[{\"FileSystemId\":\"fs-123\"}]}")
		case "/2015-02-01/mount-targets":
			if got := r.URL.Query().Get("FileSystemId"); got != "fs-123" {
				t.Errorf("FileSystemId query = %q, want %q", got, "fs-123")
			}
			writeEfsJSON(w, http.StatusInternalServerError, "{\"message\":\"temporary failure\"}")
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	generator := &EfsGenerator{}
	err := generator.loadFileSystem(newTestEfsClient(server), true, true, true)
	if err == nil {
		t.Fatal("expected mount target lookup error")
	}
	if !strings.Contains(err.Error(), "describe efs mount targets for fs-123") {
		t.Fatalf("loadFileSystem error = %q, want mount target context", err)
	}
}

func TestEfsLoadFileSystemSkipsMissingPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2015-02-01/file-systems":
			writeEfsJSON(w, http.StatusOK, "{\"FileSystems\":[{\"FileSystemId\":\"fs-123\"}]}")
		case "/2015-02-01/mount-targets":
			if got := r.URL.Query().Get("FileSystemId"); got != "fs-123" {
				t.Errorf("FileSystemId query = %q, want %q", got, "fs-123")
			}
			writeEfsJSON(w, http.StatusOK, "{\"MountTargets\":[]}")
		case "/2015-02-01/file-systems/fs-123/policy":
			w.Header().Set("x-amzn-ErrorType", "PolicyNotFound")
			writeEfsJSON(w, http.StatusNotFound, "{\"__type\":\"PolicyNotFound\",\"message\":\"policy not found\"}")
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	generator := &EfsGenerator{}
	if err := generator.loadFileSystem(newTestEfsClient(server), true, true, true); err != nil {
		t.Fatalf("loadFileSystem returned error for missing policy: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(generator.Resources))
	}
	if got := generator.Resources[0].InstanceInfo.Type; got != "aws_efs_file_system" {
		t.Fatalf("resource type = %q, want aws_efs_file_system", got)
	}
}

func TestEfsLoadFileSystemPaginatesMountTargets(t *testing.T) {
	mountTargetRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2015-02-01/file-systems":
			writeEfsJSON(w, http.StatusOK, "{\"FileSystems\":[{\"FileSystemId\":\"fs-123\"}]}")
		case "/2015-02-01/mount-targets":
			if got := r.URL.Query().Get("FileSystemId"); got != "fs-123" {
				t.Errorf("FileSystemId query = %q, want %q", got, "fs-123")
			}
			mountTargetRequests++
			switch mountTargetRequests {
			case 1:
				writeEfsJSON(w, http.StatusOK, "{\"MountTargets\":[{\"MountTargetId\":\"mt-1\"}],\"NextMarker\":\"page-2\"}")
			case 2:
				if got := r.URL.Query().Get("Marker"); got != "page-2" {
					t.Errorf("Marker query = %q, want %q", got, "page-2")
				}
				writeEfsJSON(w, http.StatusOK, "{\"MountTargets\":[{\"MountTargetId\":\"mt-2\"}]}")
			default:
				t.Errorf("unexpected mount target request %d", mountTargetRequests)
				writeEfsJSON(w, http.StatusOK, "{\"MountTargets\":[]}")
			}
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	generator := &EfsGenerator{}
	if err := generator.loadFileSystem(newTestEfsClient(server), false, true, false); err != nil {
		t.Fatalf("loadFileSystem returned error: %v", err)
	}
	if mountTargetRequests != 2 {
		t.Fatalf("mount target requests = %d, want 2", mountTargetRequests)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("len(Resources) = %d, want 2", len(generator.Resources))
	}
	for i, wantID := range []string{"mt-1", "mt-2"} {
		if got := generator.Resources[i].InstanceState.ID; got != wantID {
			t.Fatalf("resource[%d] ID = %q, want %q", i, got, wantID)
		}
	}
}

func TestEfsResourceConstructors(t *testing.T) {
	tests := []struct {
		name       string
		resource   terraformResourceResult
		wantID     string
		wantName   string
		wantType   string
		wantAttr   map[string]string
		wantExists bool
	}{
		{
			name:       "backup policy",
			resource:   newTerraformResourceResult(newEFSBackupPolicyResource("fs-123", efstypes.StatusEnabled)),
			wantID:     "fs-123",
			wantName:   terraformutils.TfSanitize("fs-123"),
			wantType:   efsBackupPolicyResourceType,
			wantAttr:   map[string]string{"file_system_id": "fs-123", "backup_policy.0.status": "ENABLED"},
			wantExists: true,
		},
		{
			name:       "backup policy empty file system",
			resource:   newTerraformResourceResult(newEFSBackupPolicyResource("", efstypes.StatusEnabled)),
			wantExists: false,
		},
		{
			name:       "backup policy transient status",
			resource:   newTerraformResourceResult(newEFSBackupPolicyResource("fs-123", efstypes.StatusEnabling)),
			wantExists: false,
		},
		{
			name: "replication configuration",
			resource: newTerraformResourceResult(newEFSReplicationConfigurationResource(efstypes.ReplicationConfigurationDescription{
				SourceFileSystemId: aws.String("fs-source"),
				Destinations: []efstypes.Destination{
					{FileSystemId: aws.String("fs-destination"), Status: efstypes.ReplicationStatusEnabled},
				},
			}, "us-east-1")),
			wantID:     "fs-source",
			wantName:   terraformutils.TfSanitize("fs-source"),
			wantType:   efsReplicationConfigurationResourceType,
			wantAttr:   map[string]string{"source_file_system_id": "fs-source"},
			wantExists: true,
		},
		{
			name: "replication configuration destination region uses canonical source",
			resource: newTerraformResourceResult(newEFSReplicationConfigurationResource(efstypes.ReplicationConfigurationDescription{
				SourceFileSystemId: aws.String("fs-source"),
				Destinations: []efstypes.Destination{
					{FileSystemId: aws.String("fs-destination"), Region: aws.String("us-west-2"), Status: efstypes.ReplicationStatusEnabled},
				},
			}, "us-west-2")),
			wantID:     "fs-source",
			wantName:   terraformutils.TfSanitize("fs-source"),
			wantType:   efsReplicationConfigurationResourceType,
			wantAttr:   map[string]string{"source_file_system_id": "fs-source"},
			wantExists: true,
		},
		{
			name: "replication configuration skips transient first destination",
			resource: newTerraformResourceResult(newEFSReplicationConfigurationResource(efstypes.ReplicationConfigurationDescription{
				SourceFileSystemId: aws.String("fs-source"),
				Destinations: []efstypes.Destination{
					{FileSystemId: aws.String("fs-deleting"), Region: aws.String("us-east-1"), Status: efstypes.ReplicationStatusDeleting},
					{FileSystemId: aws.String("fs-destination"), Region: aws.String("us-west-2"), Status: efstypes.ReplicationStatusEnabled},
				},
			}, "us-west-2")),
			wantID:     "fs-source",
			wantName:   terraformutils.TfSanitize("fs-source"),
			wantType:   efsReplicationConfigurationResourceType,
			wantAttr:   map[string]string{"source_file_system_id": "fs-source"},
			wantExists: true,
		},
		{
			name: "replication configuration skips when no destination is importable",
			resource: newTerraformResourceResult(newEFSReplicationConfigurationResource(efstypes.ReplicationConfigurationDescription{
				SourceFileSystemId: aws.String("fs-source"),
				Destinations: []efstypes.Destination{
					{FileSystemId: aws.String("fs-other"), Region: aws.String("us-east-1"), Status: efstypes.ReplicationStatusDeleting},
					{FileSystemId: aws.String("fs-destination"), Region: aws.String("us-west-2"), Status: efstypes.ReplicationStatusDeleting},
				},
			}, "us-west-2")),
			wantExists: false,
		},
		{
			name: "replication configuration falls back to source with first importable destination",
			resource: newTerraformResourceResult(newEFSReplicationConfigurationResource(efstypes.ReplicationConfigurationDescription{
				SourceFileSystemId: aws.String("fs-source"),
				Destinations: []efstypes.Destination{
					{FileSystemId: aws.String("fs-deleting"), Status: efstypes.ReplicationStatusDeleting},
					{FileSystemId: aws.String("fs-destination"), Status: efstypes.ReplicationStatusPaused},
				},
			}, "")),
			wantID:     "fs-source",
			wantName:   terraformutils.TfSanitize("fs-source"),
			wantType:   efsReplicationConfigurationResourceType,
			wantAttr:   map[string]string{"source_file_system_id": "fs-source"},
			wantExists: true,
		},
		{
			name: "replication configuration missing source",
			resource: newTerraformResourceResult(newEFSReplicationConfigurationResource(efstypes.ReplicationConfigurationDescription{
				Destinations: []efstypes.Destination{
					{FileSystemId: aws.String("fs-destination"), Status: efstypes.ReplicationStatusEnabled},
				},
			}, "us-east-1")),
			wantExists: false,
		},
		{
			name: "replication configuration transient status",
			resource: newTerraformResourceResult(newEFSReplicationConfigurationResource(efstypes.ReplicationConfigurationDescription{
				SourceFileSystemId: aws.String("fs-source"),
				Destinations: []efstypes.Destination{
					{FileSystemId: aws.String("fs-destination"), Status: efstypes.ReplicationStatusDeleting},
				},
			}, "us-east-1")),
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resource.ok != tt.wantExists {
				t.Fatalf("resource exists = %t, want %t", tt.resource.ok, tt.wantExists)
			}
			if !tt.wantExists {
				return
			}
			resource := tt.resource.resource
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
			for key, want := range tt.wantAttr {
				if got := resource.InstanceState.Attributes[key]; got != want {
					t.Fatalf("attribute %s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestEfsReplicationConfigurationMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "typed replication not found", err: &efstypes.ReplicationNotFound{}, want: true},
		{name: "api error code", err: &smithy.GenericAPIError{Code: "ReplicationNotFound"}, want: true},
		{name: "policy not found", err: &efstypes.PolicyNotFound{}, want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
		{name: "nil error", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := efsReplicationConfigurationMissing(tt.err); got != tt.want {
				t.Fatalf("efsReplicationConfigurationMissing() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEfsFileSystemPolicyMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "typed policy not found", err: &efstypes.PolicyNotFound{}, want: true},
		{name: "wrapped policy not found", err: errors.Join(errors.New("lookup failed"), &efstypes.PolicyNotFound{}), want: true},
		{name: "api error code", err: &smithy.GenericAPIError{Code: "PolicyNotFound"}, want: true},
		{name: "access denied", err: &smithy.GenericAPIError{Code: "AccessDeniedException"}, want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
		{name: "nil error", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := efsFileSystemPolicyMissing(tt.err); got != tt.want {
				t.Fatalf("efsFileSystemPolicyMissing() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEfsBackupPolicyUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "typed policy not found", err: &efstypes.PolicyNotFound{}, want: true},
		{name: "wrapped policy not found", err: errors.Join(errors.New("lookup failed"), &efstypes.PolicyNotFound{}), want: true},
		{name: "typed validation", err: &efstypes.ValidationException{}, want: true},
		{name: "wrapped validation", err: errors.Join(errors.New("lookup failed"), &efstypes.ValidationException{}), want: true},
		{name: "api error validation", err: &smithy.GenericAPIError{Code: "ValidationException"}, want: true},
		{name: "access denied", err: &smithy.GenericAPIError{Code: "AccessDeniedException"}, want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
		{name: "nil error", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := efsBackupPolicyUnavailable(tt.err); got != tt.want {
				t.Fatalf("efsBackupPolicyUnavailable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func newTestEfsClient(server *httptest.Server) *efs.Client {
	config := aws.Config{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider("test", "test", ""),
		HTTPClient:       server.Client(),
		RetryMaxAttempts: 1,
	}
	return efs.NewFromConfig(config, func(options *efs.Options) {
		options.BaseEndpoint = aws.String(server.URL)
	})
}

func writeEfsJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
