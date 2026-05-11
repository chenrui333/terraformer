// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ivstypes "github.com/aws/aws-sdk-go-v2/service/ivs/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	testIvsChannelARN                = "arn:aws:ivs:us-east-1:123456789012:channel/abc123"
	testIvsRecordingConfigurationARN = "arn:aws:ivs:us-east-1:123456789012:recording-configuration/def456"
)

func TestNewIvsChannelResource(t *testing.T) {
	resource, ok := newIvsChannelResource(&ivstypes.Channel{
		Arn:  aws.String(testIvsChannelARN),
		Name: aws.String("core-channel"),
	})
	assertIvsResource(t, resource, ok, testIvsChannelARN, ivsResourceName("channel", "core-channel", testIvsChannelARN), ivsChannelResourceType)

	if _, ok := newIvsChannelResource(nil); ok {
		t.Fatal("nil channel should be skipped")
	}
	if _, ok := newIvsChannelResource(&ivstypes.Channel{Name: aws.String("core-channel")}); ok {
		t.Fatal("channel with empty ARN should be skipped")
	}
}

func TestNewIvsRecordingConfigurationResource(t *testing.T) {
	recordingConfiguration := validIvsRecordingConfiguration("core-recording")
	resource, ok := newIvsRecordingConfigurationResource(recordingConfiguration)
	assertIvsResource(t, resource, ok, testIvsRecordingConfigurationARN, ivsResourceName("recording-configuration", "core-recording", testIvsRecordingConfigurationARN), ivsRecordingConfigurationResourceType)
	assertIvsAttribute(t, resource, "destination_configuration.#", "1")
	assertIvsAttribute(t, resource, "destination_configuration.0.s3.#", "1")
	assertIvsAttribute(t, resource, "destination_configuration.0.s3.0.bucket_name", "recording-bucket")
}

func TestNewIvsRecordingConfigurationResourceSkipsIncompleteConfigurations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ivstypes.RecordingConfiguration)
	}{
		{name: "empty ARN", mutate: func(configuration *ivstypes.RecordingConfiguration) { configuration.Arn = nil }},
		{name: "empty state", mutate: func(configuration *ivstypes.RecordingConfiguration) { configuration.State = "" }},
		{name: "missing destination", mutate: func(configuration *ivstypes.RecordingConfiguration) { configuration.DestinationConfiguration = nil }},
		{name: "missing S3 destination", mutate: func(configuration *ivstypes.RecordingConfiguration) { configuration.DestinationConfiguration.S3 = nil }},
		{name: "empty bucket", mutate: func(configuration *ivstypes.RecordingConfiguration) {
			configuration.DestinationConfiguration.S3.BucketName = nil
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configuration := validIvsRecordingConfiguration("core-recording")
			tt.mutate(configuration)
			if _, ok := newIvsRecordingConfigurationResource(configuration); ok {
				t.Fatal("incomplete recording configuration should be skipped")
			}
		})
	}

	if _, ok := newIvsRecordingConfigurationResource(nil); ok {
		t.Fatal("nil recording configuration should be skipped")
	}
}

func TestIvsARNImportID(t *testing.T) {
	if got := ivsARNImportID(testIvsChannelARN); got != testIvsChannelARN {
		t.Fatalf("IVS ARN import ID = %q, want %q", got, testIvsChannelARN)
	}
}

func TestIvsRecordingConfigurationStateImportable(t *testing.T) {
	tests := []struct {
		name  string
		state ivstypes.RecordingConfigurationState
		want  bool
	}{
		{name: "creating", state: ivstypes.RecordingConfigurationStateCreating, want: true},
		{name: "create failed", state: ivstypes.RecordingConfigurationStateCreateFailed, want: true},
		{name: "active", state: ivstypes.RecordingConfigurationStateActive, want: true},
		{name: "empty", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ivsRecordingConfigurationStateImportable(tt.state); got != tt.want {
				t.Fatalf("ivsRecordingConfigurationStateImportable(%q) = %t, want %t", tt.state, got, tt.want)
			}
		})
	}
}

func TestIvsResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(ivsResourceName("channel", "a/b_c", "d"))
	right := terraformutils.TfSanitize(ivsResourceName("channel", "a", "b_c/d"))
	if left == right {
		t.Fatalf("IVS resource names collide: %q", left)
	}
}

func TestIvsResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &ivstypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &ivstypes.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ivsResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("ivsResourceNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func validIvsRecordingConfiguration(name string) *ivstypes.RecordingConfiguration {
	return &ivstypes.RecordingConfiguration{
		Arn:   aws.String(testIvsRecordingConfigurationARN),
		Name:  aws.String(name),
		State: ivstypes.RecordingConfigurationStateActive,
		DestinationConfiguration: &ivstypes.DestinationConfiguration{
			S3: &ivstypes.S3DestinationConfiguration{
				BucketName: aws.String("recording-bucket"),
			},
		},
	}
}

func assertIvsResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
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

func assertIvsAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}
