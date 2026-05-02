// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efstypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/smithy-go"
)

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
	err := generator.loadFileSystem(newTestEfsClient(server))
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
	if err := generator.loadFileSystem(newTestEfsClient(server)); err != nil {
		t.Fatalf("loadFileSystem returned error for missing policy: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(generator.Resources))
	}
	if got := generator.Resources[0].InstanceInfo.Type; got != "aws_efs_file_system" {
		t.Fatalf("resource type = %q, want aws_efs_file_system", got)
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
